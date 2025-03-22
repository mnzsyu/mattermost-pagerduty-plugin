// server/plugin.go
package main

import (
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"

	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/client"
	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/command"
	"github.com/mnzsyu/mattermost-pagerduty-plugin/server/store/kvstore"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// kvstore is the client used to read/write KV records for this plugin.
	kvstore kvstore.KVStore

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// commandHandler is the handler for slash commands.
	commandHandler command.Command

	// pdClient is the PagerDuty API client.
	pdClient *client.PagerDutyClient

	// botUserID is the ID of the bot user.
	botUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	// Initialize KV store client
	p.kvstore = kvstore.NewKVStore(p.client)

	// Try to ensure bot exists, but continue even if it fails
	botUserID, err := p.ensureBotExists()
	if err != nil {
		p.API.LogWarn("Failed to create bot user, continuing without bot", "error", err.Error())
		// Use a default user ID or fall back to system
		p.botUserID = ""
	} else {
		p.botUserID = botUserID
	}

	// Initialize PagerDuty client
	if err := p.initializePagerDutyClient(); err != nil {
		return errors.Wrap(err, "failed to initialize PagerDuty client")
	}

	// Register slash commands - still useful even without bot
	p.commandHandler = command.NewCommandHandler(p.client, p.pdClient, p.botUserID, "com.github.mnzsyu.mattermost-pagerduty-plugin")
	if err := p.commandHandler.Register(); err != nil {
		return errors.Wrap(err, "failed to register commands")
	}

	return nil
}

// ensureBotExists ensures the bot account exists
func (p *Plugin) ensureBotExists() (string, error) {
	bot := &model.Bot{
		Username:    "pagerduty",
		DisplayName: "PagerDuty",
		Description: "A bot account for PagerDuty integration",
	}

	botUserID, err := p.client.Bot.EnsureBot(bot)
	if err != nil {
		return "", errors.Wrap(err, "failed to ensure bot user")
	}

	return botUserID, nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	// Perform any cleanup here
	return nil
}

// ExecuteCommand executes slash commands
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	response, err := p.commandHandler.Handle(args)
	if err != nil {
		return nil, model.NewAppError("ExecuteCommand", "pagerduty.command.execute.app_error", nil, err.Error(), 500)
	}
	return response, nil
}
