// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {WebSocketEvents} from 'mattermost-redux/constants';

import {id as pluginId} from './manifest';

// WebSocket event constants
export const WEBSOCKET_INCIDENT_UPDATED = 'incident_updated';
export const WEBSOCKET_INCIDENT_CREATED = 'incident_created';

/**
 * Register WebSocket event handlers
 * @param {Function} dispatch - Redux dispatch function
 */
export function registerPluginWebSocketEvent(dispatch) {
    const handleEvent = (msg) => {
        const {event, data} = msg.data;

        switch (event) {
        case WEBSOCKET_INCIDENT_UPDATED:
            handleIncidentUpdated(dispatch, data);
            break;
        case WEBSOCKET_INCIDENT_CREATED:
            handleIncidentCreated(dispatch, data);
            break;
        }
    };

    // Register the WebSocket event handler with the client
    window.WebSocket.addEventListener(`${WebSocketEvents.PLUGIN_CUSTOM}/${pluginId}`, handleEvent);
}

/**
 * Handle incident updated event
 */
function handleIncidentUpdated() {
    // Handle incident updates here
    // This would typically dispatch a Redux action to update the client state
    // or possibly trigger a notification

    // This will be implemented in a future iteration as needed
}

/**
 * Handle incident created event
 */
function handleIncidentCreated() {
    // Handle new incident creation here
    // This would typically dispatch a Redux action to update the client state
    // or possibly trigger a notification

    // This will be implemented in a future iteration as needed
}

// Add newline at end of file
