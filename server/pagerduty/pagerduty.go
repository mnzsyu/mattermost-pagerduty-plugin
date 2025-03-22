// server/pagerduty/pagerduty.go
package pagerduty

import (
	"time"
)

// Incident represents a PagerDuty incident
type Incident struct {
	ID                 string           `json:"id"`
	IncidentNumber     int              `json:"incident_number"`
	Title              string           `json:"title"`
	Description        string           `json:"description"`
	Status             string           `json:"status"`
	Urgency            string           `json:"urgency"`
	CreatedAt          time.Time        `json:"created_at"`
	Service            Service          `json:"service"`
	Assignments        []Assignment     `json:"assignments"`
	LastStatusChangeBy User             `json:"last_status_change_by,omitempty"`
	LastStatusChangeAt time.Time        `json:"last_status_change_at,omitempty"`
	AlertCount         int              `json:"alert_count,omitempty"`
	HTMLURL            string           `json:"html_url"`
	EscalationPolicy   EscalationPolicy `json:"escalation_policy"`
}

// EscalationPolicy represents a PagerDuty escalation policy
type EscalationPolicy struct {
	ID      string `json:"id"`
	Name    string `json:"summary"`
	HTMLURL string `json:"html_url"`
}

// Service represents a PagerDuty service
type Service struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Assignment represents a PagerDuty incident assignment
type Assignment struct {
	Assignee User      `json:"assignee"`
	At       time.Time `json:"at"`
}

// User represents a PagerDuty user
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// WebhookPayload represents the payload from PagerDuty webhook
type WebhookPayload struct {
	Messages []WebhookMessage `json:"messages"`
}

// V3WebhookPayload represents a PagerDuty V3 webhook payload
type V3WebhookPayload struct {
	Event V3Event `json:"event"`
}

// V3Event represents a PagerDuty V3 webhook event
type V3Event struct {
	ID           string      `json:"id"`
	EventType    string      `json:"event_type"`
	ResourceType string      `json:"resource_type"`
	OccurredAt   string      `json:"occurred_at"`
	Agent        V3Reference `json:"agent"`
	Data         Incident    `json:"data"`
}

// V3Reference represents a PagerDuty V3 reference object
type V3Reference struct {
	HTMLURL string `json:"html_url"`
	ID      string `json:"id"`
	Self    string `json:"self"`
	Summary string `json:"summary"`
	Type    string `json:"type"`
}

// WebhookMessage represents a message in the PagerDuty webhook payload
type WebhookMessage struct {
	ID         string                 `json:"id"`
	Event      string                 `json:"event"`
	CreatedOn  time.Time              `json:"created_on"`
	Incident   Incident               `json:"incident"`
	LogEntries []LogEntry             `json:"log_entries,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// LogEntry represents a PagerDuty log entry
type LogEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Agent     User      `json:"agent"`
	Channel   Channel   `json:"channel"`
	Note      string    `json:"note,omitempty"`
}

// Channel represents a PagerDuty notification channel
type Channel struct {
	Type string `json:"type"`
}

// PostAttachment is used to create Mattermost post attachments for incidents
type PostAttachment struct {
	ID        string   `json:"id"`
	PostID    string   `json:"post_id"`
	ChannelID string   `json:"channel_id"`
	Incident  Incident `json:"incident"`
}

// IncidentActionPayload is the payload sent for incident actions
type IncidentActionPayload struct {
	IncidentID string `json:"incident_id"`
	Action     string `json:"action"` // acknowledge, resolve, reassign
	UserID     string `json:"user_id"`
	AssigneeID string `json:"assignee_id,omitempty"` // Only used for reassign
}

// APIResponse is a generic response from PagerDuty API
type APIResponse struct {
	Incident  *Incident  `json:"incident,omitempty"`
	Incidents []Incident `json:"incidents,omitempty"`
	Error     *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}
