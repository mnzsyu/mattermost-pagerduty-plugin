// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import IncidentPostTypeComponent from './components/incident_post_type';
import Root from './components/root';
import {id as pluginId} from './manifest';
import {registerPluginWebSocketEvent} from './websocket';

export default class Plugin {
    // eslint-disable-next-line no-unused-vars
    initialize(registry, store) {
        // Register the main component
        registry.registerRootComponent(Root);

        // Register post type for PagerDuty incidents
        registry.registerPostTypeComponent('custom_pagerduty_incident', IncidentPostTypeComponent);

        // Register slash command autocomplete
        registry.registerSlashCommandWillBePostedHook(
            (message, args) => {
                if (!message.startsWith('/pagerduty')) {
                    return {message, args};
                }

                // Handle any client-side autocomplete or validation here
                return {message, args};
            },
        );

        // Register websocket event handlers
        registerPluginWebSocketEvent(store.dispatch);
    }
}

window.registerPlugin(pluginId, new Plugin());
