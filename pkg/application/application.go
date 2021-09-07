package application

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
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
	eventBus           *cluster.ClusterEventBus
	eventsSubscription cluster.Subscription
	functionWorkers    []*sandbox.FunctionExecutionWorker

	// API
	apiToken  string
	dataStore store.Store
}

type eventSubscription struct {
	eventPattern     string
	subscriptionFunc eventbus.SubscriptionFunc
}

func NewApplication(cfg *config.Config, appName string, s store.Store, ceb *cluster.ClusterEventBus) (*Application, error) {
	apiToken := util.TokenGenerator()
	app := &Application{
		config:          cfg,
		appName:         appName,
		dataStore:       s,
		eventBus:        ceb,
		apiToken:        apiToken,
		functionWorkers: []*sandbox.FunctionExecutionWorker{},
	}

	var err error

	app.eventsSubscription, err = app.eventBus.QueueSubscribe("events", "events.workers", func(msg *nats.Msg) {
		var eventData cluster.PublishEvent
		if err := json.Unmarshal(msg.Data, &eventData); err != nil {
			log.Errorf("Could not unmarshal event: %s - %s", err, string(msg.Data))
			return
		}
		if funcsToInvoke, ok := app.definitions.Events[eventData.Name]; ok {
			for _, funcToInvoke := range funcsToInvoke {
				resp, err := app.InvokeFunction(string(funcToInvoke), eventData.Data)
				if err != nil {
					log.Errorf("Error invoking %s: %s", funcToInvoke, err)
				}
				if resp != nil && msg.Reply != "" {
					if err := msg.Respond(util.MustJsonByteSlice(resp)); err != nil {
						log.Error("Could not respond to event")
					}
				}
			}
		}
	})
	if err != nil {
		return nil, errors.Wrap(err, "event subscribe")
	}

	return app, nil
}

// Only for testing
func NewMockApplication(config *config.Config, appName string) *Application {
	return &Application{
		appName:         appName,
		definitions:     &definition.Definitions{},
		dataStore:       &store.MockStore{},
		config:          config,
		functionWorkers: []*sandbox.FunctionExecutionWorker{},
	}
}

var FunctionDoesNotExistError = errors.New("function does not exist")

func (app *Application) InvokeFunction(name string, event interface{}) (interface{}, error) {
	return app.eventBus.InvokeFunction(name, event)
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

	fmt.Println(defs.Markdown())

	app.reset()

	apiURL := fmt.Sprintf("http://%s:%d/%s", "%s", app.config.APIBindPort, app.appName)
	for name, def := range defs.Functions {
		worker, err := sandbox.NewFunctionExecutionWorker(app.config, apiURL, app.apiToken, app.eventBus, string(name), def.Config, def.Code)
		if err != nil {
			log.Errorf("Could not spin up function worker for %s: %s", name, err)
		}
		app.functionWorkers = append(app.functionWorkers, worker)
	}

	// log.Info("Starting jobs...")
	// for name, def := range defs.Jobs {
	// 	jobTimeoutCtx, cancel := context.WithTimeout(context.Background(), app.config.SanboxJobInitTimeout)
	// 	job, err := app.sandbox.Job(jobTimeoutCtx, string(name), def.Config, def.Code)
	// 	if err != nil {
	// 		cancel()
	// 		return errors.Wrap(err, "init job")
	// 	}
	// 	cancel()
	// 	jobStartTimeoutCtx, cancel := context.WithTimeout(context.Background(), app.config.SandboxJobStartTimeout)
	// 	if err := job.Start(jobStartTimeoutCtx); err != nil {
	// 		cancel()
	// 		return errors.Wrap(err, "start job")
	// 	}
	// 	cancel()
	// }

	log.Info("Ready to go.")
	app.eventBus.Publish("init", []byte{})
	return nil
}

// reset but ready to start again
func (app *Application) reset() {
	// stop all function workers
	for _, worker := range app.functionWorkers {
		if err := worker.Close(); err != nil {
			log.Errorf("Could not close worker: %s", err)
		}
	}
	app.functionWorkers = []*sandbox.FunctionExecutionWorker{}
}

func (app *Application) Close() error {
	if err := app.eventsSubscription.Unsubscribe(); err != nil {
		return err
	}
	app.reset()
	return app.dataStore.Close()
}

func (app *Application) Definitions() *definition.Definitions {
	return app.definitions
}

func (app *Application) EventBus() *cluster.ClusterEventBus {
	return app.eventBus
}
