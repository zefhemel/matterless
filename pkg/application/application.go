package application

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
	"os"
	"time"
)

type Application struct {
	// Definitions
	appName     string
	definitions *definition.Definitions
	code        string

	// Runtime
	sandbox            *sandbox.Sandbox
	eventBus           eventbus.EventBus
	eventSubscriptions []eventSubscription

	// API
	apiToken    string
	dataStore   store.Store
	cfg         *config.Config
	appDataPath string
}

type eventSubscription struct {
	eventPattern     string
	subscriptionFunc eventbus.SubscriptionFunc
}

func NewApplication(cfg *config.Config, appName string) (*Application, error) {
	appDataPath := fmt.Sprintf("%s/%s", cfg.DataDir, util.SafeFilename(appName))
	if err := os.MkdirAll(appDataPath, 0700); err != nil {
		return nil, errors.Wrap(err, "create data dir")
	}
	dataStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/store", appDataPath))
	if err != nil {
		return nil, errors.Wrap(err, "create data store dir")
	}
	eventBus := eventbus.NewLocalEventBus()
	app := &Application{
		cfg:         cfg,
		appDataPath: appDataPath,
		appName:     appName,
		eventBus:    eventBus,
		dataStore:   dataStore,
		apiToken:    util.TokenGenerator(),
		// TODO: Make this configurable
		sandbox: sandbox.NewSandbox(fmt.Sprintf("http://%s:%d/%s", "%s", cfg.APIBindPort, appName), eventBus, 1*time.Minute, 5*time.Minute),
	}

	return app, nil
}

func (app *Application) LoadFromDisk() error {
	applicationFile := fmt.Sprintf("%s/application.md", app.appDataPath)
	if _, err := os.Stat(applicationFile); err == nil {
		buf, err := os.ReadFile(applicationFile)
		if err != nil {
			return errors.Wrap(err, "reading application file")
		}
		return app.Eval(string(buf))
	} else {
		return err
	}

	return nil
}

// Only for testing
func NewMockApplication(appName string) *Application {
	return &Application{
		appName:            appName,
		definitions:        &definition.Definitions{},
		sandbox:            sandbox.NewSandbox(fmt.Sprintf("http://localhost/%s", appName), eventbus.NewLocalEventBus(), 1*time.Minute, 5*time.Minute),
		dataStore:          &store.MockStore{},
		eventBus:           eventbus.NewLocalEventBus(),
		eventSubscriptions: []eventSubscription{},
	}
}

func (app *Application) InvokeFunction(name definition.FunctionID, event interface{}) interface{} {
	functionDef := app.definitions.Functions[name]
	log.Debug("Now triggering event to ", name)
	// TODO: Remove hardcoded values
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	functionInstance, err := app.sandbox.Function(ctx, string(name), app.definitions.Config, app.definitions.ModulesForLanguage(functionDef.Language), functionDef.Config, functionDef.Code)
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
	for envName, envVal := range app.cfg.GlobalEnv {
		defs.Config[envName] = envVal
	}

	defs.Normalize()
	if err := defs.Desugar(); err != nil {
		return err
	}

	app.definitions = defs

	app.extendEnviron()
	app.reset()

	for eventName, funcs := range defs.Events {
		// Copy variable into loop scope (closure)
		funcsToInvoke := funcs
		sub := eventSubscription{
			eventPattern: eventName,
			subscriptionFunc: func(eventName string, eventData interface{}) {
				for _, funcToInvoke := range funcsToInvoke {
					app.InvokeFunction(funcToInvoke, eventData)
				}
			},
		}
		app.eventBus.Subscribe(eventName, sub.subscriptionFunc)
		app.eventSubscriptions = append(app.eventSubscriptions, sub)
	}

	log.Info("Starting jobs...")
	timeOutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for name, def := range defs.Jobs {
		ji, err := app.sandbox.Job(timeOutCtx, string(name), app.definitions.Config, app.definitions.ModulesForLanguage(def.Language), def.Config, def.Code)
		if err != nil {
			return errors.Wrap(err, "init job")
		}
		if err := ji.Start(timeOutCtx); err != nil {
			return errors.Wrap(err, "init job")
		}
	}
	if err := os.WriteFile(fmt.Sprintf("%s/application.md", app.appDataPath), []byte(code), 0600); err != nil {
		log.Errorf("Could not write application.md file to disk: %s", err)
	}

	log.Info("Ready to go.")
	return nil
}

// reset but ready to start again
func (app *Application) reset() {
	// Unsubscribe all event listeners
	for _, subscription := range app.eventSubscriptions {
		app.eventBus.Unsubscribe(subscription.eventPattern, subscription.subscriptionFunc)
	}
	app.eventSubscriptions = make([]eventSubscription, 0, 10)

	// Flush the sandbox
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
