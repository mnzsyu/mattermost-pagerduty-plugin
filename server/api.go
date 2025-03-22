package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// ServeHTTP handles HTTP requests to the plugin
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()

	// API router for protected endpoints (require Mattermost authentication)
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(p.MattermostAuthorizationRequired)

	// Add the hello endpoint for testing
	apiRouter.HandleFunc("/hello", p.handleHello).Methods(http.MethodGet)

	// Handler for incident actions
	apiRouter.HandleFunc("/incidents/{incident_id}/acknowledge", p.handleAcknowledge).Methods(http.MethodPost)
	apiRouter.HandleFunc("/incidents/{incident_id}/resolve", p.handleResolve).Methods(http.MethodPost)
	apiRouter.HandleFunc("/incidents/{incident_id}/reassign", p.handleReassign).Methods(http.MethodPost)

	// Endpoints for commands
	apiRouter.HandleFunc("/incidents", p.handleListIncidents).Methods(http.MethodGet)
	apiRouter.HandleFunc("/incidents/{incident_id}", p.handleGetIncident).Methods(http.MethodGet)

	// PagerDuty webhook endpoint (not protected by authentication)
	router.HandleFunc("/webhook", p.HandleWebhook).Methods(http.MethodPost)

	router.ServeHTTP(w, r)
}

// MattermostAuthorizationRequired is middleware that ensures the request has a valid Mattermost user
func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHello is a simple hello world handler for testing
func (p *Plugin) handleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello, world!"))
}

// handleAcknowledge handles acknowledging an incident
func (p *Plugin) handleAcknowledge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	incidentID := vars["incident_id"]
	if incidentID == "" {
		http.Error(w, "Missing incident ID", http.StatusBadRequest)
		return
	}

	p.HandleIncidentAction(w, r, incidentID, ActionAcknowledge)
}

// handleResolve handles resolving an incident
func (p *Plugin) handleResolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	incidentID := vars["incident_id"]
	if incidentID == "" {
		http.Error(w, "Missing incident ID", http.StatusBadRequest)
		return
	}

	p.HandleIncidentAction(w, r, incidentID, ActionResolve)
}

// handleReassign handles reassigning an incident
func (p *Plugin) handleReassign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	incidentID := vars["incident_id"]
	if incidentID == "" {
		http.Error(w, "Missing incident ID", http.StatusBadRequest)
		return
	}

	p.HandleIncidentAction(w, r, incidentID, ActionReassign)
}

// handleListIncidents handles listing incidents (for slash command)
func (p *Plugin) handleListIncidents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Get incidents from PagerDuty
	incidents, err := p.pdClient.ListIncidents(query)
	if err != nil {
		p.API.LogError("Failed to list incidents", "error", err.Error())
		http.Error(w, "Failed to list incidents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the incidents
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Create response
	response := struct {
		Incidents []interface{} `json:"incidents"`
	}{
		Incidents: make([]interface{}, len(incidents)),
	}

	for i, incident := range incidents {
		response.Incidents[i] = incident
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		p.API.LogError("Failed to encode incidents", "error", err.Error())
		http.Error(w, "Failed to encode incidents", http.StatusInternalServerError)
		return
	}
}

// handleGetIncident handles getting a single incident (for slash command)
func (p *Plugin) handleGetIncident(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	incidentID := vars["incident_id"]
	if incidentID == "" {
		http.Error(w, "Missing incident ID", http.StatusBadRequest)
		return
	}

	// Get incident from PagerDuty
	incident, err := p.pdClient.GetIncident(incidentID)
	if err != nil {
		p.API.LogError("Failed to get incident", "error", err.Error())
		http.Error(w, "Failed to get incident: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the incident
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(incident); err != nil {
		p.API.LogError("Failed to encode incident", "error", err.Error())
		http.Error(w, "Failed to encode incident", http.StatusInternalServerError)
		return
	}
}
