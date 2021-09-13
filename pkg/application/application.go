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
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
)

type Application struct {
	// Definitions
	appName     string
	definitions *definition.Definitions

	// Runtime
	config                  *config.Config
	eventBus                *cluster.ClusterEventBus
	eventsSubscription      cluster.Subscription
	startWorkerSubscription cluster.Subscription
	sandbox                 *sandbox.Sandbox

	// API
	apiToken  string
	dataStore store.Store
}

func NewApplication(cfg *config.Config, appName string, s store.Store, ceb *cluster.ClusterEventBus) (*Application, error) {
	apiURL := fmt.Sprintf("http://%s:%d/%s", "%s", cfg.APIBindPort, appName)
	apiToken := util.TokenGenerator()
	sb, err := sandbox.NewSandbox(cfg, apiURL, apiToken, ceb)
	if err != nil {
		return nil, errors.Wrap(err, "sandbox create")
	}
	app := &Application{
		config:   cfg,
		appName:  appName,
		eventBus: ceb,
		apiToken: apiToken,
		sandbox:  sb,
	}

	app.dataStore = store.NewEventedStore(s, func(key string, val interface{}) {
		if err := app.PublishAppEvent(fmt.Sprintf("store:put:%s", key), map[string]interface{}{
			"key":       key,
			"new_value": val,
		}); err != nil {
			log.Errorf("Could not publish store:put event: %s", err)
		}
	}, func(key string) {
		if err := app.PublishAppEvent(fmt.Sprintf("store:del:%s", key), map[string]interface{}{
			"key": key,
		}); err != nil {
			log.Errorf("Could not publish store:del event: %s", err)
		}
	})

	// We send all app events through a single event queue
	app.eventsSubscription, err = app.eventBus.QueueSubscribe(cluster.EventPublishEvent, "event.workers", func(msg *nats.Msg) {
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

	app.startWorkerSubscription, err = app.eventBus.SubscribeRequestJobWorker(func(jobName string) {
		job := app.definitions.Jobs[definition.FunctionID(jobName)]
		log.Info("Starting job worker ", jobName)
		if err := app.sandbox.StartJobWorker(definition.FunctionID(jobName), job.Config, job.Code); err != nil {
			log.Errorf("Could not start job %s: %s", jobName, err)
		}
	})

	return app, nil
}

// Only for testing
func NewMockApplication(config *config.Config, appName string) *Application {
	return &Application{
		appName:     appName,
		definitions: &definition.Definitions{},
		dataStore:   &store.MockStore{},
		config:      config,
	}
}

var FunctionDoesNotExistError = errors.New("function does not exist")

func (app *Application) InvokeFunction(name string, event interface{}) (interface{}, error) {
	return app.eventBus.InvokeFunction(name, event)
}

// Publish a (custom) application event
func (app *Application) PublishAppEvent(name string, event interface{}) error {
	return app.eventBus.Publish(cluster.EventPublishEvent, util.MustJsonByteSlice(cluster.PublishEvent{
		Name: name,
		Data: event,
	}))
}

func (app *Application) Eval(defs *definition.Definitions) error {
	app.definitions = defs
	defs.InterpolateStoreValues(app.dataStore)

	// fmt.Println(defs.Markdown())

	app.reset()

	log.Info("Loading functions...")
	for name, def := range defs.Functions {
		for i := 0; i < def.Config.Instances; i++ {
			if err := app.sandbox.StartFunctionWorker(string(name), def.Config, def.Code); err != nil {
				log.Errorf("Could not spin up function worker for %s: %s", name, err)
			}
		}
	}

	//log.Info("Loading jobs...")
	//for name, def := range defs.Jobs {
	//	if err := app.sandbox.StartJobWorker(name, def.Config, def.Code); err != nil {
	//		log.Errorf("Could not spin up job worker for %s: %s", name, err)
	//	}
	//}

	log.Info("Ready to go.")
	return nil
}

func (app *Application) EvalString(code string) error {
	defs, err := definition.Check(code, app.config)
	if err != nil {
		return err
	}

	return app.Eval(defs)
}

//func (app *Application) StartJobs() error {
//	log.Info("Starting jobs...")
//	for name, def := range app.definitions.Jobs {
//		if _, err := app.InvokeFunction(string(name), def.Config.Init); err != nil {
//			return err
//		}
//	}
//	return nil
//}

// reset but ready to start again
func (app *Application) reset() {
	app.sandbox.Flush()
}

func (app *Application) Close() error {
	if err := app.eventsSubscription.Unsubscribe(); err != nil {
		return err
	}
	if err := app.startWorkerSubscription.Unsubscribe(); err != nil {
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

func (app *Application) Sandbox() *sandbox.Sandbox {
	return app.sandbox
}

func (app *Application) Name() string {
	return app.appName
}
