package application

import (
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
	code        string
	definitions *definition.Definitions

	// Runtime
	eventSources map[string]eventsource.EventSource
	sandbox      *sandbox.DockerSandbox
	adminClient  *model.Client4

	// Callbacks
	// TODO: Consider switching to channels instead?
	logCallback func(kind string, message string)

	// API
	apiToken  string
	dataStore store.Store
	cfg       config.Config
}

func NewApplication(cfg config.Config, adminClient *model.Client4, appName string, logCallback func(kind, message string)) *Application {
	dataStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/%s", cfg.LevelDBDatabasesPath, util.SafeFilename(appName)))
	if err != nil {
		log.Fatal("Could not create data store for ", appName)
	}
	return &Application{
		cfg:          cfg,
		appName:      appName,
		adminClient:  adminClient,
		eventSources: map[string]eventsource.EventSource{},
		logCallback:  logCallback,
		// TODO: Make this configurable
		sandbox:   sandbox.NewDockerSandbox(30*time.Second, 1*time.Minute),
		dataStore: dataStore,
		apiToken:  util.TokenGenerator(),
	}
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
	result, log, err := app.sandbox.Invoke(event, app.definitions.CompileFunctionCode(functionDef.Code), app.definitions.Environment)
	if err != nil {
		app.logCallback(string(name), err.Error())
	}
	if log != "" {
		app.logCallback(string(name), log)
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

	results := definition.Check(defs)
	if results.String() != "" {
		// Error while checking
		return errors.Wrap(errors.New(results.String()), "declaration check")
	}

	app.extendEnviron()

	log.Debug("Starting listeners...")
	app.Stop()

	// Rebuild the whole thing
	for name, def := range defs.MattermostClients {
		mmSource, err := eventsource.NewMatterMostSource(name, def, app.InvokeFunction)
		if err != nil {
			return err
		}
		app.eventSources[name] = mmSource
		mmSource.Start()
		mmSource.ExtendDefinitions(defs)
		log.Debug("Starting Mattermost client: ", name)
	}

	for name, def := range defs.Bots {
		botSource, err := eventsource.NewBotSource(app.adminClient, name, def, app.InvokeFunction)
		if err != nil {
			return err
		}
		app.eventSources[name] = botSource
		botSource.Start()
		botSource.ExtendDefinitions(defs)
		log.Debug("Starting Bot: ", name)
	}

	// Not processing API Gateways here, done externally

	// Slash commands
	for name, def := range defs.SlashCommands {
		scmd := eventsource.NewSlashCommandSource(app.cfg, app.adminClient, app.appName, name, def)
		if err != nil {
			return err
		}
		app.eventSources[name] = scmd
		log.Debug("Starting Slashcommand: ", name)
		scmd.Start()
		scmd.ExtendDefinitions(defs)
	}

	for name, def := range defs.Crons {
		c := eventsource.NewCronSource(def, app.InvokeFunction)
		if err != nil {
			return err
		}
		app.eventSources[name] = c
		log.Debug("Starting cron: ", name)
		c.Start()
		c.ExtendDefinitions(defs)
	}

	log.Debug("Testing functions...")
	testResults := definition.TestDeclarations(defs, app.sandbox)
	for name, functionResult := range testResults.Functions {
		if functionResult.Logs != "" {
			app.logCallback(string(name), functionResult.Logs)
		}
	}
	if testResults.String() != "" {
		return errors.Wrap(errors.New(testResults.String()), "test run")
	}
	log.Debug("All ready to go!")
	return nil
}

func (app *Application) Stop() {
	// First, stop all event sources
	for sourceName, eventSource := range app.eventSources {
		log.Debug("Stopping ", sourceName)
		eventSource.Stop()
	}
	// Then stop all functions in sandbox
	log.Debug("Stopping sandbox")
	app.sandbox.FlushAll()
}

func (app *Application) Close() error {
	app.Stop()
	return app.dataStore.Close()
}

func (app *Application) Definitions() *definition.Definitions {
	return app.definitions
}
