package command

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/client"
	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/pagerduty"
)

// Constants for slash commands
const (
	CommandPagerDuty = "pagerduty"
	SubCommandList   = "list"
	SubCommandOnCall = "oncall"
	SubCommandGet    = "get"
	SubCommandHelp   = "help"
)

// Handler handles PagerDuty slash commands
type Handler struct {
	client        *pluginapi.Client
	pdClient      *client.PagerDutyClient
	botUserID     string
	pluginURLPath string
}

// Command is the interface for slash command handling
type Command interface {
	Register() error
	Handle(args *model.CommandArgs) (*model.CommandResponse, error)
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(client *pluginapi.Client, pdClient *client.PagerDutyClient, botUserID string, pluginID string) Command {
	return &Handler{
		client:        client,
		pdClient:      pdClient,
		botUserID:     botUserID,
		pluginURLPath: fmt.Sprintf("/plugins/%s", pluginID),
	}
}

// Register registers the slash commands
func (h *Handler) Register() error {
	// Register the main command
	if err := h.client.SlashCommand.Register(&model.Command{
		Trigger:          CommandPagerDuty,
		AutoComplete:     true,
		AutoCompleteDesc: "Interact with PagerDuty",
		AutoCompleteHint: "[command]",
		DisplayName:      "PagerDuty",
		Description:      "Integration with PagerDuty",
	}); err != nil {
		return err
	}

	return nil
}

// Handle handles slash command execution
func (h *Handler) Handle(args *model.CommandArgs) (*model.CommandResponse, error) {
	// Split the command arguments
	fields := strings.Fields(args.Command)
	if len(fields) < 2 {
		return h.helpCommand(args), nil
	}

	// Get subcommand
	subcommand := fields[1]

	switch strings.ToLower(subcommand) {
	case SubCommandList:
		additionalArgs := []string{}
		if len(fields) > 2 {
			additionalArgs = fields[2:]
		}
		return h.listIncidentsCommand(args, additionalArgs), nil
	case SubCommandOnCall:
		return h.onCallCommand(args), nil
	case SubCommandGet:
		if len(fields) < 3 {
			return &model.CommandResponse{
				ResponseType: model.CommandResponseTypeEphemeral,
				Text:         "Please provide an incident ID or number",
			}, nil
		}
		return h.getIncidentCommand(args, fields[2]), nil
	case SubCommandHelp:
		return h.helpCommand(args), nil
	default:
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Unknown subcommand: %s. Try `/pagerduty help` for available commands.", subcommand),
		}, nil
	}
}

// listIncidentsCommand handles listing incidents
func (h *Handler) listIncidentsCommand(args *model.CommandArgs, params []string) *model.CommandResponse {
	// Parse options
	options := url.Values{}
	options.Set("limit", "10") // Default limit

	// Parse additional parameters
	var status, service, urgency string

	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "limit":
			if limit, err := strconv.Atoi(value); err == nil && limit > 0 && limit <= 25 {
				options.Set("limit", value)
			}
		case "status":
			status = value
			options.Set("statuses[]", value)
		case "service":
			service = value
			options.Set("service_ids[]", value)
		case "urgency":
			urgency = value
			options.Set("urgencies[]", value)
		}
	}

	// Get incidents from PagerDuty
	incidents, err := h.pdClient.ListIncidents(options)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Error getting incidents: %s", err.Error()),
		}
	}

	// Filter incidents if necessary
	var filteredIncidents []pagerduty.Incident
	for _, incident := range incidents {
		if (status == "" || incident.Status == status) &&
			(service == "" || incident.Service.ID == service) &&
			(urgency == "" || incident.Urgency == urgency) {
			filteredIncidents = append(filteredIncidents, incident)
		}
	}

	// Format response
	text := "### PagerDuty Incidents\n\n"
	if len(filteredIncidents) == 0 {
		text += "No incidents found matching your criteria."
	} else {
		text += "| # | Status | Service | Title | Assigned To |\n"
		text += "| --- | --- | --- | --- | --- |\n"

		for _, incident := range filteredIncidents {
			// Format assignees
			assignees := "Unassigned"
			if len(incident.Assignments) > 0 {
				var names []string
				for _, assignment := range incident.Assignments {
					names = append(names, assignment.Assignee.Name)
				}
				assignees = strings.Join(names, ", ")
			}

			// Format status
			status := cases.Title(language.English).String(incident.Status)

			// Format service
			service := incident.Service.Name

			// Add row
			text += fmt.Sprintf("| [#%d](%s) | %s | %s | %s | %s |\n",
				incident.IncidentNumber,
				incident.HTMLURL,
				status,
				service,
				incident.Title,
				assignees,
			)
		}
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         text,
	}
}

// onCallCommand handles getting on-call information
func (h *Handler) onCallCommand(args *model.CommandArgs) *model.CommandResponse {
	// Here we would fetch on-call schedule from PagerDuty
	// For now, respond with a placeholder message
	text := "### PagerDuty On-Call Information\n\n"
	text += "On-call schedule not implemented yet. Please check the PagerDuty web interface for on-call information."

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}

// getIncidentCommand handles getting a single incident
func (h *Handler) getIncidentCommand(args *model.CommandArgs, incidentIdentifier string) *model.CommandResponse {
	// Get incident from PagerDuty
	var incident *pagerduty.Incident
	var err error

	// Check if incident identifier is a number (incident number) or string (incident ID)
	if incidentNumber, numErr := strconv.Atoi(incidentIdentifier); numErr == nil {
		// It's an incident number, get all incidents and filter
		options := url.Values{}
		options.Set("incident_number", strconv.Itoa(incidentNumber))

		incidents, listErr := h.pdClient.ListIncidents(options)
		if listErr != nil {
			return &model.CommandResponse{
				ResponseType: model.CommandResponseTypeEphemeral,
				Text:         fmt.Sprintf("Error getting incident: %s", listErr.Error()),
			}
		}

		if len(incidents) == 0 {
			return &model.CommandResponse{
				ResponseType: model.CommandResponseTypeEphemeral,
				Text:         fmt.Sprintf("No incident found with number: %d", incidentNumber),
			}
		}

		incident = &incidents[0]
	} else {
		// It's an incident ID
		incident, err = h.pdClient.GetIncident(incidentIdentifier)
		if err != nil {
			return &model.CommandResponse{
				ResponseType: model.CommandResponseTypeEphemeral,
				Text:         fmt.Sprintf("Error getting incident: %s", err.Error()),
			}
		}
	}

	// Format response
	text := fmt.Sprintf("### PagerDuty Incident #%d: %s\n\n", incident.IncidentNumber, incident.Title)
	text += fmt.Sprintf("**Status:** %s\n", cases.Title(language.English).String(incident.Status))
	text += fmt.Sprintf("**Urgency:** %s\n", cases.Title(language.English).String(incident.Urgency))
	text += fmt.Sprintf("**Service:** %s\n", incident.Service.Name)

	// Format assignees
	if len(incident.Assignments) > 0 {
		var names []string
		for _, assignment := range incident.Assignments {
			names = append(names, assignment.Assignee.Name)
		}
		text += fmt.Sprintf("**Assigned To:** %s\n", strings.Join(names, ", "))
	} else {
		text += "**Assigned To:** Unassigned\n"
	}

	// Format dates
	text += fmt.Sprintf("**Created:** %s\n", incident.CreatedAt.Format(time.RFC3339))
	if !incident.LastStatusChangeAt.IsZero() {
		text += fmt.Sprintf("**Last Status Change:** %s\n", incident.LastStatusChangeAt.Format(time.RFC3339))
	}

	// Add description
	text += "\n**Description:**\n"
	text += incident.Description

	// Add link
	text += fmt.Sprintf("\n\n[View in PagerDuty](%s)", incident.HTMLURL)

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         text,
	}
}

// helpCommand shows the help information
func (h *Handler) helpCommand(args *model.CommandArgs) *model.CommandResponse {
	text := "### PagerDuty Command Help\n\n"
	text += "* `/pagerduty list [status=triggered|acknowledged|resolved] [urgency=high|low] [limit=5]` - List incidents\n"
	text += "* `/pagerduty get <incident_id_or_number>` - Get details for a specific incident\n"
	text += "* `/pagerduty oncall` - Show who is currently on call\n"
	text += "* `/pagerduty help` - Show this help message\n"

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}
