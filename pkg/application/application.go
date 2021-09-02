package application

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
)

type Application struct {
	// Definitions
	appName     string
	definitions *definition.Definitions
	code        string

	// Runtime
	config             *config.Config
	sandbox            *sandbox.Sandbox
	eventBus           eventbus.EventBus
	eventSubscriptions []eventSubscription
	appDataPath        string

	// API
	apiToken  string
	dataStore store.Store
}

type eventSubscription struct {
	eventPattern     string
	subscriptionFunc eventbus.SubscriptionFunc
}

func NewApplication(cfg *config.Config, appName string) (*Application, error) {
	appDataPath := fmt.Sprintf("%s/%s", cfg.DataDir, util.SafeFilename(appName))
	eventBus := eventbus.NewLocalEventBus()

	if err := os.MkdirAll(appDataPath, 0700); err != nil {
		return nil, errors.Wrap(err, "create data dir")
	}
	levelDBStore, err := store.NewLevelDBStore(fmt.Sprintf("%s/store", appDataPath))
	if err != nil {
		return nil, errors.Wrap(err, "create data store dir")
	}
	dataStore := store.NewEventedStore(levelDBStore, eventBus)

	apiToken := util.TokenGenerator()

	sb, err := sandbox.NewSandbox(cfg, fmt.Sprintf("http://%s:%d/%s", "%s", cfg.APIBindPort, appName), apiToken, eventBus)
	if err != nil {
		return nil, errors.Wrap(err, "sandbox init")
	}
	app := &Application{
		config:      cfg,
		appDataPath: appDataPath,
		appName:     appName,
		eventBus:    eventBus,
		dataStore:   dataStore,
		apiToken:    apiToken,
		sandbox:     sb,
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
}

// Only for testing
func NewMockApplication(config *config.Config, appName string) *Application {
	sb, err := sandbox.NewSandbox(config, fmt.Sprintf("http://localhost/%s", appName), "1234", eventbus.NewLocalEventBus())
	if err != nil {
		log.Fatal(err)
	}
	return &Application{
		appName:            appName,
		definitions:        &definition.Definitions{},
		sandbox:            sb,
		dataStore:          &store.MockStore{},
		eventBus:           eventbus.NewLocalEventBus(),
		eventSubscriptions: []eventSubscription{},
		config:             config,
	}
}

var FunctionDoesNotExistError = errors.New("function does not exist")

func (app *Application) InvokeFunction(name definition.FunctionID, event interface{}) (interface{}, error) {
	functionDef, ok := app.definitions.Functions[name]
	if !ok {
		return nil, FunctionDoesNotExistError
	}
	log.Debug("Now invoking function ", name)

	ctx, cancel := context.WithTimeout(context.Background(), app.config.FunctionRunTimeout)
	defer cancel()

	functionInstance, err := app.sandbox.Function(ctx, string(name), functionDef.Config, functionDef.Code)
	if err != nil {
		return nil, errors.Wrap(err, "function init")
	}
	result, err := functionInstance.Invoke(ctx, event)
	if err != nil {
		return nil, errors.Wrap(err, "function invoke")
	}
	return result, nil
}

func (app *Application) Eval(code string) error {
	log.Debug("Parsing and checking definitions...")
	app.code = code
	defs, err := definition.Parse(code)
	if err != nil {
		return err
	}

	if err := defs.InlineImports(fmt.Sprintf("%s/.importcache", app.config.DataDir)); err != nil {
		return err
	}
	if err := defs.ExpandMacros(); err != nil {
		return err
	}

	app.definitions = defs
	app.interpolateStoreValues()

	//fmt.Println(defs.Markdown())

	app.reset()

	for eventName, funcs := range defs.Events {
		// Copy variable into loop scope (closure)
		funcsToInvoke := funcs
		sub := eventSubscription{
			eventPattern: eventName,
			subscriptionFunc: func(eventName string, eventData interface{}) {
				for _, funcToInvoke := range funcsToInvoke {
					_, err := app.InvokeFunction(funcToInvoke, eventData)
					if err != nil {
						app.EventBus().Publish(fmt.Sprintf("logs:%s", funcToInvoke), sandbox.LogEntry{
							FunctionName: string(funcToInvoke),
							Message:      fmt.Sprintf("[%s Error] %s", funcToInvoke, err.Error()),
						})
					}
				}
			},
		}
		app.eventBus.Subscribe(eventName, sub.subscriptionFunc)
		app.eventSubscriptions = append(app.eventSubscriptions, sub)
	}

	log.Info("Starting jobs...")
	for name, def := range defs.Jobs {
		jobTimeoutCtx, cancel := context.WithTimeout(context.Background(), app.config.SanboxJobInitTimeout)
		job, err := app.sandbox.Job(jobTimeoutCtx, string(name), def.Config, def.Code)
		if err != nil {
			cancel()
			return errors.Wrap(err, "init job")
		}
		cancel()
		jobStartTimeoutCtx, cancel := context.WithTimeout(context.Background(), app.config.SandboxJobStartTimeout)
		if err := job.Start(jobStartTimeoutCtx); err != nil {
			cancel()
			return errors.Wrap(err, "start job")
		}
		cancel()
	}

	if app.config.PersistApps {
		if err := os.WriteFile(fmt.Sprintf("%s/application.md", app.appDataPath), []byte(code), 0600); err != nil {
			log.Errorf("Could not write application.md file to disk: %s", err)
		}
	}

	log.Info("Ready to go.")
	app.eventBus.Publish("init", struct{}{})
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
