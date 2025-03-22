# Mattermost PagerDuty Integration Plugin

A Mattermost plugin for PagerDuty integration that provides interactive incident management directly from Mattermost.

## Features

- Real-time incident notifications in Mattermost
- Interactive buttons to acknowledge and resolve incidents
- Ability to reassign incidents to other users
- Slash commands to view and manage incidents
- Incident status updates shown directly in the channel

## Installation

1. Download the latest release from the [GitHub Releases page](https://github.com/mnzsyu/mattermost-pagerduty-plugin/releases)
2. Upload the plugin to your Mattermost instance via System Console → Plugin Management
3. Configure the plugin with your PagerDuty API key and other settings

## Configuration

1. Go to System Console → Plugins → PagerDuty
2. Enter your PagerDuty API Key (General Access API key from PagerDuty)
3. (Optional) Enter a Webhook Secret if you're configuring a secured webhook in PagerDuty
4. Specify the default channel for incident notifications (without the `~` prefix)
5. Save the configuration and enable the plugin

## Setting up PagerDuty Webhooks

1. In PagerDuty, go to Integrations → Generic Webhooks
2. Create a new webhook
3. Set the webhook URL to: `https://your-mattermost-instance.com/plugins/com.github.mnzsyu.mattermost-pagerduty-plugin/webhook`
4. (Optional) Set a webhook secret and add the same secret to the plugin configuration in Mattermost
5. Select the events you want to receive (recommended: all incident events)

## Usage

### Slash Commands

- `/pagerduty list [status=triggered|acknowledged|resolved] [urgency=high|low] [limit=5]` - List incidents
- `/pagerduty get <incident_id_or_number>` - Get details for a specific incident
- `/pagerduty oncall` - Show who is currently on call
- `/pagerduty help` - Show help information

### Interactive Actions

Incident notifications include interactive buttons:

- **Acknowledge** - Mark an incident as acknowledged
- **Resolve** - Mark an incident as resolved
- **Reassign** - Reassign an incident to another user

## Development

### Prerequisites

- Go 1.22 or higher
- Node.js 16 or higher
- npm 8 or higher

### Building

```bash
make
```

This will create a distribution in the `dist/` folder that can be uploaded to Mattermost.

### Development Workflow

```bash
# Run in development mode with hot reloading
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=your-token
make watch
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.