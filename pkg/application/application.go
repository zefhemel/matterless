package application

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
	"time"
)

type Application struct {
	// Definitions
	appName     string
	definitions *definition.Definitions
	code        string

	// Runtime
	connectedEventSources []eventsource.EventSource
	sandbox               *sandbox.DockerSandbox
	adminClient           *model.Client4

	// API
	apiToken  string
	dataStore store.Store
	cfg       config.Config
}

func NewApplication(cfg config.Config, adminClient *model.Client4, appName string) *Application {
	dataStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/%s", cfg.LevelDBDatabasesPath, util.SafeFilename(appName)))
	if err != nil {
		log.Fatal("Could not create data store for ", appName)
	}
	app := &Application{
		cfg:         cfg,
		appName:     appName,
		adminClient: adminClient,
		dataStore:   dataStore,
		apiToken:    util.TokenGenerator(),
		// TODO: Make this configurable
		sandbox: sandbox.NewDockerSandbox(1*time.Minute, 5*time.Minute),
	}

	return app
}

// Logs returns a channel of all log messages coming from all running functions
func (app *Application) Logs() chan sandbox.LogEntry {
	return app.sandbox.Logs()
}

// Only for testing
func NewMockApplication(appName string, defs *definition.Definitions) *Application {
	return &Application{
		appName:     appName,
		definitions: defs,
	}
}

func (app *Application) InvokeFunction(name definition.FunctionID, event interface{}) interface{} {
	functionDef := app.definitions.Functions[name]
	log.Debug("Now triggering event to ", name)
	// TODO: Remove hardcoded values
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	functionInstance, err := app.sandbox.Function(ctx, string(name), app.definitions.Environment, app.definitions.ModulesForLanguage(functionDef.Language), functionDef.Code)
	if err != nil {
		app.Logs() <- sandbox.LogEntry{
			Instance: functionInstance,
			Message:  fmt.Sprintf("[%s Error] %s", name, err.Error()),
		}
		return nil
	}
	result, err := functionInstance.Invoke(ctx, event)
	if err != nil {
		app.Logs() <- sandbox.LogEntry{
			Instance: functionInstance,
			Message:  fmt.Sprintf("[%s Error] %s", name, err.Error()),
		}
	}
	return result
}

func (app *Application) CurrentCode() string {
	return app.code
}

func (app *Application) Eval(code string) error {
	log.Debug("Parsing and checking definitions...")
	app.code = code
	defs, err := definition.Parse(code)
	if err != nil {
		return err
	}
	defs.Normalize()

	app.definitions = defs

	app.extendEnviron()

	log.Debug("Starting listeners...")
	app.reset()

	// Rebuild the whole thing
	for name, def := range defs.MattermostClients {
		src, err := eventsource.NewMatterMostSource(name, def, app.InvokeFunction)
		if err != nil {
			return err
		}
		app.connectedEventSources = append(app.connectedEventSources, src)
		src.ExtendDefinitions(defs)
		log.Debug("Starting Mattermost client: ", name)
	}

	for name, def := range defs.Bots {
		src, err := eventsource.NewBotSource(app.adminClient, name, def, app.InvokeFunction)
		if err != nil {
			return err
		}
		app.connectedEventSources = append(app.connectedEventSources, src)
		src.ExtendDefinitions(defs)
		log.Debug("Starting Bot: ", name)
	}

	for name, def := range defs.SlashCommands {
		src, err := eventsource.NewSlashCommandSource(app.cfg, app.adminClient, app.appName, name, def)
		if err != nil {
			return err
		}
		app.connectedEventSources = append(app.connectedEventSources, src)
		log.Debug("Starting Slashcommand: ", name)
		src.ExtendDefinitions(defs)
	}

	c := eventsource.NewCronSource(defs.Crons, app.InvokeFunction)
	if err != nil {
		return err
	}
	app.connectedEventSources = append(app.connectedEventSources, c)
	log.Debug("Starting cron")
	c.ExtendDefinitions(defs)

	log.Debug("Testing functions...")
	testResults := definition.TestDeclarations(defs, app.sandbox)
	for name, functionResult := range testResults.Functions {
		if functionResult != nil {
			app.Logs() <- sandbox.LogEntry{
				Instance: nil,
				Message:  fmt.Sprintf("[%s Check error] %s", name, functionResult.Error()),
			}
		}
	}
	if testResults.String() != "" {
		return errors.Wrap(errors.New(testResults.String()), "test run")
	}
	log.Debug("All ready to go!")
	return nil
}

// reset but ready to start again
func (app *Application) reset() {
	// First, stop all event sources
	for _, eventSource := range app.connectedEventSources {
		eventSource.Close()
	}
	app.connectedEventSources = []eventsource.EventSource{}
	app.sandbox.Flush()
}

func (app *Application) Close() error {
	app.reset()
	app.sandbox.Close()
	return app.dataStore.Close()
}

func (app *Application) Definitions() *definition.Definitions {
	return app.definitions
}
