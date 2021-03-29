package application

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
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
	eventBus              eventbus.EventBus

	// API
	apiToken  string
	dataStore store.Store
	cfg       config.Config
}

func NewApplication(cfg config.Config, appName string) *Application {
	dataStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/%s", cfg.LevelDBDatabasesPath, util.SafeFilename(appName)))
	if err != nil {
		log.Fatal("Could not create data store for ", appName)
	}
	eventBus := eventbus.NewLocalEventBus()
	app := &Application{
		cfg:       cfg,
		appName:   appName,
		eventBus:  eventBus,
		dataStore: dataStore,
		apiToken:  util.TokenGenerator(),
		// TODO: Make this configurable
		sandbox: sandbox.NewDockerSandbox(eventBus, 1*time.Minute, 5*time.Minute),
	}

	return app
}

// Only for testing
func NewMockApplication(appName string) *Application {
	return &Application{
		appName:     appName,
		definitions: &definition.Definitions{},
		sandbox:     sandbox.NewDockerSandbox(eventbus.NewLocalEventBus(), 1*time.Minute, 5*time.Minute),
		dataStore:   &store.MockStore{},
		eventBus:    eventbus.NewLocalEventBus(),
	}
}

func (app *Application) InvokeFunction(name definition.FunctionID, event interface{}) interface{} {
	functionDef := app.definitions.Functions[name]
	log.Debug("Now triggering event to ", name)
	// TODO: Remove hardcoded values
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	functionInstance, err := app.sandbox.Function(ctx, string(name), app.definitions.Environment, app.definitions.ModulesForLanguage(functionDef.Language), functionDef.Config, functionDef.Code)
	if err != nil {
		app.EventBus().Publish(fmt.Sprintf("logs:%s", name), sandbox.LogEntry{
			Instance: functionInstance,
			Message:  fmt.Sprintf("[%s Error] %s", name, err.Error()),
		})
		return nil
	}
	result, err := functionInstance.Invoke(ctx, event)
	if err != nil {
		app.EventBus().Publish(fmt.Sprintf("logs:%s", name), sandbox.LogEntry{
			Instance: functionInstance,
			Message:  fmt.Sprintf("[%s Error] %s", name, err.Error()),
		})
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

	app.reset()

	es := eventsource.NewEventEventSource(app.eventBus, defs.Events, app.InvokeFunction)
	app.connectedEventSources = append(app.connectedEventSources, es)
	es.ExtendDefinitions(defs)

	log.Info("Starting jobs...")
	for name, def := range defs.Jobs {
		ji, err := app.sandbox.Job(context.Background(), string(name), app.definitions.Environment, app.definitions.ModulesForLanguage(def.Language), def.Config, def.Code)
		if err != nil {
			return errors.Wrap(err, "init job")
		}
		envMap, err := ji.Start(context.Background())
		if err != nil {
			return errors.Wrap(err, "init job")
		}
		for k, v := range envMap {
			app.definitions.Environment[k] = v
		}
	}

	//log.Info("Initializing functions...")
	//for name, def := range defs.Functions {
	//	_, err := app.sandbox.Function(context.Background(), string(name), app.definitions.Environment, app.definitions.ModulesForLanguage(def.Language), def.Config, def.Code)
	//	if err != nil {
	//		return errors.Wrap(err, "init function")
	//	}
	//}

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

func (app *Application) EventBus() eventbus.EventBus {
	return app.eventBus
}
