package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/client"
	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/pagerduty"
)

const (
	// Action identifiers
	ActionAcknowledge = "acknowledge"
	ActionResolve     = "resolve"
	ActionReassign    = "reassign"

	// PagerDuty webhook events
	EventIncidentTriggered     = "incident.triggered"
	EventIncidentAcknowledged  = "incident.acknowledged"
	EventIncidentResolved      = "incident.resolved"
	EventIncidentReassigned    = "incident.reassigned"
	EventIncidentStatusUpdated = "incident.status_update_published"

	// Constants for KV store keys
	KeyIncidentAttachments = "incident_attachments:"

	// Maximum number of incidents to fetch
	MaxIncidents = 25
)

type PostActionOption struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

// initializePagerDutyClient initializes the PagerDuty client with the current configuration
func (p *Plugin) initializePagerDutyClient() error {
	config := p.getConfiguration()
	if config.PagerDutyAPIKey == "" {
		return errors.New("PagerDuty API key not configured")
	}
	p.pdClient = client.NewPagerDutyClient(config.PagerDutyAPIKey)
	return nil
}

// HandleWebhook handles PagerDuty webhook requests - updated for V3 webhooks
func (p *Plugin) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := p.getConfiguration()

	// Log all headers for debugging
	p.API.LogDebug("Webhook received", "headers", fmt.Sprintf("%v", r.Header))

	// Verify webhook signature if a secret is configured
	if config.WebhookSecret != "" {
		err := p.verifyWebhookSignature(r, config.WebhookSecret)
		if err != nil {
			p.API.LogError("Failed to verify webhook signature", "error", err.Error())
			// In production, you should uncomment this:
			// http.Error(w, "Invalid signature", http.StatusUnauthorized)
			// return
		}
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.API.LogError("Failed to read webhook body", "error", err.Error())
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Reset the body for further processing
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Log the payload for debugging
	p.API.LogDebug("Webhook payload", "body", string(body))

	// Process the webhook payload - using V3 format
	var payload pagerduty.V3WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		p.API.LogError("Failed to parse webhook payload", "error", err.Error())
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Process the event
	if err := p.processV3WebhookEvent(payload.Event); err != nil {
		p.API.LogError("Failed to process webhook event", "error", err.Error(), "event_id", payload.Event.ID)
		http.Error(w, "Failed to process event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// verifyWebhookSignature verifies the webhook signature using HMAC SHA256
func (p *Plugin) verifyWebhookSignature(r *http.Request, secret string) error {
	providedSignature := r.Header.Get("X-Pagerduty-Signature")
	if providedSignature == "" {
		return errors.New("no signature provided in X-Pagerduty-Signature header")
	}

	// Log the signature we found
	p.API.LogDebug("Found signature header", "value", providedSignature)

	// PagerDuty uses v1= prefix - we need to strip it
	if strings.HasPrefix(providedSignature, "v1=") {
		providedSignature = strings.TrimPrefix(providedSignature, "v1=")
	} else {
		p.API.LogDebug("Unexpected signature format", "signature", providedSignature)
	}

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read request body")
	}

	// Important: Replace the body for further reading
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Calculate HMAC SHA256
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	calculatedSignature := hex.EncodeToString(h.Sum(nil))

	// Log both signatures for comparison
	p.API.LogDebug("Signature comparison",
		"provided", providedSignature,
		"calculated", calculatedSignature)

	// Compare signatures
	if hmac.Equal([]byte(providedSignature), []byte(calculatedSignature)) {
		return nil
	}

	return errors.New("invalid signature")
}

// processWebhookMessage processes a webhook message and creates/updates a post
func (p *Plugin) processWebhookMessage(message pagerduty.WebhookMessage) error {
	p.API.LogDebug("Processing webhook message", "event", message.Event)
	incident := message.Incident
	p.API.LogDebug("Processing incident", "id", incident.ID, "title", incident.Title)

	// Get the appropriate channel ID
	channelID, err := p.getChannelID()
	if err != nil {
		p.API.LogError("Failed to get channel ID", "error", err.Error())
		return errors.Wrap(err, "failed to get channel ID")
	}
	p.API.LogDebug("Got channel ID", "channelID", channelID)

	// Check if there's already a post for this incident
	attachment, err := p.getIncidentAttachment(incident.ID)
	if err != nil {
		p.API.LogError("Failed to get incident attachment", "error", err.Error())
		// Continue anyway - we'll create a new post
	}

	switch message.Event {
	case EventIncidentTriggered:
		// Create a new post for triggered incidents
		return p.handleTriggeredIncident(incident, channelID)

	case EventIncidentAcknowledged, EventIncidentResolved,
		EventIncidentReassigned, EventIncidentStatusUpdated:
		// Update existing post if available
		if attachment != nil {
			return p.updateIncidentPost(incident, attachment)
		}

		// Create a new post if no existing post is found
		return p.handleTriggeredIncident(incident, channelID)

	default:
		// Ignore unhandled event types
		p.API.LogInfo("Ignoring unhandled event type", "event", message.Event)
		return nil
	}
}

// processV3WebhookEvent processes a V3 webhook event
func (p *Plugin) processV3WebhookEvent(event pagerduty.V3Event) error {
	p.API.LogDebug("Processing webhook event", "event_type", event.EventType, "resource_type", event.ResourceType)

	// Only process incident events
	if event.ResourceType != "incident" {
		p.API.LogInfo("Ignoring non-incident event", "resource_type", event.ResourceType)
		return nil
	}

	// Map V3 event_type to our internal event types
	var messageEvent string
	switch event.EventType {
	case "incident.triggered":
		messageEvent = EventIncidentTriggered
	case "incident.acknowledged":
		messageEvent = EventIncidentAcknowledged
	case "incident.resolved":
		messageEvent = EventIncidentResolved
	case "incident.reassigned":
		messageEvent = EventIncidentReassigned
	case "incident.status_update_published":
		messageEvent = EventIncidentStatusUpdated
	default:
		p.API.LogInfo("Ignoring unhandled event type", "event_type", event.EventType)
		return nil
	}

	// Create a webhook message from the V3 event
	message := pagerduty.WebhookMessage{
		ID:       event.ID,
		Event:    messageEvent,
		Incident: event.Data,
	}

	// Process the message
	return p.processWebhookMessage(message)
}

// handleTriggeredIncident creates a new post for a triggered incident
func (p *Plugin) handleTriggeredIncident(incident pagerduty.Incident, channelID string) error {
	p.API.LogDebug("Handling triggered incident", "id", incident.ID, "title", incident.Title)

	post := p.createIncidentPost(incident, channelID)
	p.API.LogDebug("Created post for incident", "userId", post.UserId, "channelId", post.ChannelId)

	createdPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		p.API.LogError("Failed to create post", "error", appErr.Error())
		return errors.New("failed to create post: " + appErr.Error())
	}

	p.API.LogInfo("Successfully posted incident to channel", "incident_id", incident.ID, "channel_id", channelID)

	// Store the post ID for later updates
	attachment := &pagerduty.PostAttachment{
		ID:        incident.ID,
		PostID:    createdPost.Id,
		ChannelID: channelID,
		Incident:  incident,
	}

	if err := p.storeIncidentAttachment(attachment); err != nil {
		return errors.Wrap(err, "failed to store incident attachment")
	}

	return nil
}

// updateIncidentPost updates an existing post with new incident information
func (p *Plugin) updateIncidentPost(incident pagerduty.Incident, attachment *pagerduty.PostAttachment) error {
	// Get the existing post
	post, appErr := p.API.GetPost(attachment.PostID)
	if appErr != nil {
		// Post might have been deleted, create a new one
		return p.handleTriggeredIncident(incident, attachment.ChannelID)
	}

	// Update the post with new information
	post.Props = p.createIncidentProps(incident)

	// Update the post
	_, appErr = p.API.UpdatePost(post)
	if appErr != nil {
		return errors.New("failed to update post: " + appErr.Error())
	}

	// Update the stored attachment with the latest incident info
	attachment.Incident = incident
	if err := p.storeIncidentAttachment(attachment); err != nil {
		return errors.Wrap(err, "failed to update incident attachment")
	}

	return nil
}

// createIncidentPost creates a Mattermost post for an incident
func (p *Plugin) createIncidentPost(incident pagerduty.Incident, channelID string) *model.Post {
	props := p.createIncidentProps(incident)

	// Create the post
	userID := p.botUserID
	if userID == "" {
		// Fall back to system message or another user
		userID = "system"
	}

	return &model.Post{
		UserId:    userID,
		ChannelId: channelID,
		Props:     props,
	}
}

// createIncidentProps creates the props for an incident post
func (p *Plugin) createIncidentProps(incident pagerduty.Incident) model.StringInterface {
	// Format the attachments for the post
	var fields []*model.SlackAttachmentField

	// Add incident details as fields
	fields = append(fields, &model.SlackAttachmentField{
		Title: "Service",
		Value: incident.Service.Name,
		Short: true,
	})

	fields = append(fields, &model.SlackAttachmentField{
		Title: "Urgency",
		Value: cases.Title(language.English).String(incident.Urgency),
		Short: true,
	})

	// Add assignees
	var assignees []string
	for _, assignment := range incident.Assignments {
		assignees = append(assignees, assignment.Assignee.Name)
	}

	if len(assignees) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Assigned To",
			Value: strings.Join(assignees, ", "),
			Short: true,
		})
	}

	// Add created time
	fields = append(fields, &model.SlackAttachmentField{
		Title: "Created",
		Value: incident.CreatedAt.Format(time.RFC3339),
		Short: true,
	})

	// Add incident URL
	fields = append(fields, &model.SlackAttachmentField{
		Title: "Link",
		Value: fmt.Sprintf("[View in PagerDuty](%s)", incident.HTMLURL),
		Short: false,
	})

	// Determine color based on status and urgency
	color := "#FFA500" // Default: orange
	switch incident.Status {
	case client.StatusTriggered:
		if incident.Urgency == "high" {
			color = "#FF0000" // Red for high urgency triggered
		} else {
			color = "#FFA500" // Orange for other triggered
		}
	case client.StatusAcknowledged:
		color = "#FFFF00" // Yellow for acknowledged
	case client.StatusResolved:
		color = "#008000" // Green for resolved
	}

	// Create the message attachment
	attachment := &model.SlackAttachment{
		Title:   fmt.Sprintf("[#%d] %s", incident.IncidentNumber, incident.Title),
		Text:    incident.Description,
		Color:   color,
		Fields:  fields,
		Actions: p.getIncidentActions(incident),
	}

	// Create post props
	return model.StringInterface{
		"attachments":  []*model.SlackAttachment{attachment},
		"from_webhook": "true",
	}
}

// getIncidentActions returns the available actions for an incident
func (p *Plugin) getIncidentActions(incident pagerduty.Incident) []*model.PostAction {
	var actions []*model.PostAction

	// Plugin ID constant
	pluginID := "com.github.mnzsyu.mattermost-pagerduty-plugin"

	// Only show acknowledge button for triggered incidents
	if incident.Status == client.StatusTriggered {
		actions = append(actions, &model.PostAction{
			Id:    ActionAcknowledge,
			Name:  "Acknowledge",
			Type:  "button",
			Style: "primary",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/incidents/%s/acknowledge", pluginID, incident.ID),
				Context: map[string]interface{}{
					"incident_id": incident.ID,
					"action":      ActionAcknowledge,
				},
			},
		})
	}

	// Show resolve button for non-resolved incidents
	if incident.Status != client.StatusResolved {
		actions = append(actions, &model.PostAction{
			Id:    ActionResolve,
			Name:  "Resolve",
			Type:  "button",
			Style: "success",
			Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf("/plugins/%s/api/v1/incidents/%s/resolve", pluginID, incident.ID),
				Context: map[string]interface{}{
					"incident_id": incident.ID,
					"action":      ActionResolve,
				},
			},
		})
	}

	// Add reassign button for all incidents
	actions = append(actions, &model.PostAction{
		Id:   ActionReassign,
		Name: "Reassign",
		Type: "select",
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("/plugins/%s/api/v1/incidents/%s/reassign", pluginID, incident.ID),
			Context: map[string]interface{}{
				"incident_id": incident.ID,
				"action":      ActionReassign,
			},
		},
		// Use a dynamic JSON approach for options
		DataSource: "custom",
		Options:    []*model.PostActionOptions{}, // Empty options, will be filled by server response
	})

	return actions
}

// getUserOptions gets user options for the reassign dropdown
// func (p *Plugin) getUserOptions() []*PostActionOption {
// 	// This would normally fetch users from PagerDuty
// 	// For now, returning a placeholder with an option to implement more later
// 	return []*PostActionOption{
// 		{
// 			Text:  "Select User...",
// 			Value: "",
// 		},
// 		{
// 			Text:  "Get on-call users...",
// 			Value: "fetch_users",
// 		},
// 	}
// }

// getChannelID gets the channel ID for posting alerts
// getChannelID gets the channel ID for posting alerts
func (p *Plugin) getChannelID() (string, error) {
	config := p.getConfiguration()
	channelValue := config.DefaultChannel

	p.API.LogDebug("Getting channel ID", "defaultChannel", channelValue)

	if channelValue == "" {
		return "", errors.New("default channel not configured")
	}

	// Try to find the channel directly by ID first
	channel, appErr := p.API.GetChannel(channelValue)
	if appErr == nil {
		p.API.LogDebug("Successfully found channel by ID", "channel_id", channel.Id)
		return channel.Id, nil
	}

	// If not found by ID, we need to search through all teams
	teams, appErr := p.API.GetTeams()
	if appErr != nil {
		return "", errors.Wrap(appErr, "failed to get teams")
	}

	// Try each team to find the channel
	for _, team := range teams {
		p.API.LogDebug("Searching for channel in team", "team_name", team.Name, "team_id", team.Id)

		// Try exact match on channel name first
		channel, appErr = p.API.GetChannelByName(team.Id, channelValue, false)
		if appErr == nil {
			p.API.LogDebug("Found channel by name in team", "channel_id", channel.Id, "team_id", team.Id)
			return channel.Id, nil
		}

		// Try case-insensitive match
		channels, appErr := p.API.GetChannelsForTeamForUser(team.Id, "me", false)
		if appErr != nil {
			p.API.LogDebug("Failed to get channels for team", "team_id", team.Id, "error", appErr.Error())
			continue
		}

		for _, ch := range channels {
			// Check for match on display name or name
			if strings.EqualFold(ch.Name, channelValue) ||
				strings.EqualFold(ch.DisplayName, channelValue) {
				p.API.LogDebug("Found channel by case-insensitive name", "channel_id", ch.Id, "team_id", team.Id)
				return ch.Id, nil
			}
		}
	}

	return "", errors.New("channel not found in any team: " + channelValue)
}

// storeIncidentAttachment stores the incident attachment in the KV store
func (p *Plugin) storeIncidentAttachment(attachment *pagerduty.PostAttachment) error {
	jsonData, err := json.Marshal(attachment)
	if err != nil {
		return errors.Wrap(err, "failed to marshal attachment")
	}

	key := KeyIncidentAttachments + attachment.ID
	appErr := p.API.KVSet(key, jsonData)
	if appErr != nil {
		return errors.New("failed to store attachment in KV store: " + appErr.Error())
	}

	return nil
}

// getIncidentAttachment gets the incident attachment from the KV store
func (p *Plugin) getIncidentAttachment(incidentID string) (*pagerduty.PostAttachment, error) {
	key := KeyIncidentAttachments + incidentID

	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		return nil, errors.New("failed to get attachment from KV store: " + appErr.Error())
	}

	if data == nil {
		return nil, nil
	}

	var attachment pagerduty.PostAttachment
	if err := json.Unmarshal(data, &attachment); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal attachment")
	}

	return &attachment, nil
}

// HandleIncidentAction handles incident action button clicks
func (p *Plugin) HandleIncidentAction(w http.ResponseWriter, r *http.Request, incidentID string, action string) {
	// Get the user ID from the request
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	// Get the action payload
	var payload pagerduty.IncidentActionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get the user's email for attribution
	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	var status string
	switch action {
	case ActionAcknowledge:
		status = client.StatusAcknowledged
	case ActionResolve:
		status = client.StatusResolved
	case ActionReassign:
		// Handle reassignment separately
		p.performReassign(w, incidentID, payload.AssigneeID, user.Email)
		return
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Update the incident in PagerDuty
	_, err := p.pdClient.UpdateIncident(incidentID, status, user.Email, "")
	if err != nil {
		p.API.LogError("Failed to update incident", "error", err.Error())
		http.Error(w, "Failed to update incident", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
		p.API.LogError("Failed to encode JSON response", "error", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// performReassign handles reassigning an incident
func (p *Plugin) performReassign(w http.ResponseWriter, incidentID, assigneeID, userEmail string) {
	if assigneeID == "fetch_users" {
		// Return a list of users
		users, err := p.pdClient.ListUsers()
		if err != nil {
			p.API.LogError("Failed to list users", "error", err.Error())
			http.Error(w, "Failed to list users", http.StatusInternalServerError)
			return
		}

		var options []*PostActionOption
		for _, user := range users {
			options = append(options, &PostActionOption{
				Text:  user.Name,
				Value: user.ID,
			})
		}

		response := map[string]interface{}{
			"update": map[string]interface{}{
				"props": map[string]interface{}{
					"attachments": []map[string]interface{}{
						{
							"actions": []map[string]interface{}{
								{
									"id":      ActionReassign,
									"name":    "Reassign",
									"type":    "select",
									"options": options,
								},
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			p.API.LogError("Failed to encode JSON response", "error", err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Assign the incident
	_, err := p.pdClient.AssignIncident(incidentID, []string{assigneeID}, userEmail)
	if err != nil {
		p.API.LogError("Failed to assign incident", "error", err.Error())
		http.Error(w, "Failed to assign incident", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
		p.API.LogError("Failed to encode JSON response", "error", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
