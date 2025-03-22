package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/pagerduty"
)

const (
	pagerDutyAPIBaseURL = "https://api.pagerduty.com"

	// PagerDuty API endpoints
	incidentsEndpoint = "/incidents"
	usersEndpoint     = "/users"
	servicesEndpoint  = "/services"

	// PagerDuty incident statuses
	StatusTriggered    = "triggered"
	StatusAcknowledged = "acknowledged"
	StatusResolved     = "resolved"
)

// PagerDutyClient is the client for interacting with the PagerDuty API
type PagerDutyClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewPagerDutyClient creates a new PagerDuty API client
func NewPagerDutyClient(apiKey string) *PagerDutyClient {
	return &PagerDutyClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetIncident gets a single incident by ID
func (c *PagerDutyClient) GetIncident(incidentID string) (*pagerduty.Incident, error) {
	endpoint := fmt.Sprintf("%s%s/%s", pagerDutyAPIBaseURL, incidentsEndpoint, incidentID)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("failed to get incident: %s, status: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Incident pagerduty.Incident `json:"incident"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return &response.Incident, nil
}

// ListIncidents lists incidents with optional filters
func (c *PagerDutyClient) ListIncidents(params url.Values) ([]pagerduty.Incident, error) {
	endpoint := fmt.Sprintf("%s%s?%s", pagerDutyAPIBaseURL, incidentsEndpoint, params.Encode())

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("failed to list incidents: %s, status: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Incidents []pagerduty.Incident `json:"incidents"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return response.Incidents, nil
}

// UpdateIncident updates an incident status
func (c *PagerDutyClient) UpdateIncident(incidentID, status string, userEmail string, note string) (*pagerduty.Incident, error) {
	endpoint := fmt.Sprintf("%s%s/%s", pagerDutyAPIBaseURL, incidentsEndpoint, incidentID)

	payload := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":   "incident_reference",
			"status": status,
		},
	}

	// Add note if provided
	if note != "" {
		payload["note"] = map[string]string{
			"content": note,
		}
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	// Add From header with user email
	if userEmail != "" {
		req.Header.Set("From", userEmail)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("failed to update incident: %s, status: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Incident pagerduty.Incident `json:"incident"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return &response.Incident, nil
}

// AssignIncident assigns an incident to a user
func (c *PagerDutyClient) AssignIncident(incidentID string, userIDs []string, userEmail string) (*pagerduty.Incident, error) {
	endpoint := fmt.Sprintf("%s%s/%s", pagerDutyAPIBaseURL, incidentsEndpoint, incidentID)

	assignments := make([]map[string]interface{}, len(userIDs))
	for i, userID := range userIDs {
		assignments[i] = map[string]interface{}{
			"assignee": map[string]string{
				"id":   userID,
				"type": "user_reference",
			},
		}
	}

	payload := map[string]interface{}{
		"incident": map[string]interface{}{
			"type":        "incident_reference",
			"assignments": assignments,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	// Add From header with user email
	if userEmail != "" {
		req.Header.Set("From", userEmail)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("failed to assign incident: %s, status: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Incident pagerduty.Incident `json:"incident"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return &response.Incident, nil
}

// ListUsers lists users in the PagerDuty account
func (c *PagerDutyClient) ListUsers() ([]pagerduty.User, error) {
	endpoint := fmt.Sprintf("%s%s", pagerDutyAPIBaseURL, usersEndpoint)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("failed to list users: %s, status: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Users []pagerduty.User `json:"users"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return response.Users, nil
}

// ListServices lists services in the PagerDuty account
func (c *PagerDutyClient) ListServices() ([]pagerduty.Service, error) {
	endpoint := fmt.Sprintf("%s%s", pagerDutyAPIBaseURL, servicesEndpoint)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("failed to list services: %s, status: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Services []pagerduty.Service `json:"services"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return response.Services, nil
}

// setHeaders sets the required headers for PagerDuty API requests
func (c *PagerDutyClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")
	req.Header.Set("Authorization", "Token token="+c.apiKey)
}
