package application

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/checker"
	"github.com/zefhemel/matterless/pkg/declaration"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/util"
)

type Application struct {
	// Declarations
	declarations *declaration.Declarations

	// Runtime
	eventSources map[string]eventsource.EventSource
	sandbox      sandbox.Sandbox

	// Callbacks
	// TODO: Consider switching to channels instead?
	logCallback func(kind string, message string)
}

func NewApplication(logCallback func(kind, message string)) *Application {
	return &Application{
		eventSources: map[string]eventsource.EventSource{},
		logCallback:  logCallback,
		sandbox:      sandbox.NewNodeDockerSandbox(),
	}
}

func (app *Application) Eval(code string) error {
	decls, err := declaration.Parse(code)
	if err != nil {
		return err
	}
	results := declaration.Check(decls)
	if results.String() != "" {
		// Error while checking
		return errors.Wrap(errors.New(results.String()), "declaration check")
	}
	testResults := checker.TestDeclarations(decls, app.sandbox)
	for _, functionResult := range testResults.Functions {
		if functionResult.Logs != "" {
			app.logCallback("Function test", functionResult.Logs)
		}
	}
	if testResults.String() != "" {
		return errors.Wrap(errors.New(testResults.String()), "test run")
	}

	// First, stop all event sources
	for sourceName, eventSource := range app.eventSources {
		log.Debug("Stopping listener: ", sourceName)
		eventSource.Stop()
	}

	// Rebuild the whole thing
	for sourceName, sourceDef := range decls.Sources {
		mmSource, err := eventsource.NewMatterMostSource(sourceDef.URL, sourceDef.Token)
		if err != nil {
			return err
		}
		app.eventSources[sourceName] = mmSource
		mmSource.Start()
		log.Debug("Starting listener: ", sourceName)
		go app.eventProcessor(mmSource, decls)
	}

	return nil
}

func (app *Application) eventProcessor(source eventsource.EventSource, decls *declaration.Declarations) {
	sb := sandbox.NewNodeDockerSandbox()
	for evt := range source.Events() {
		wsEvent, ok := evt.(*model.WebSocketEvent)
		if !ok {
			log.Debug("Got non websocket event", evt)
			continue
		}

		log.Debug("Got event to process ", wsEvent)

		for _, subscriptionDef := range decls.Subscriptions {
			if util.StringSliceContains(subscriptionDef.EventTypes, wsEvent.EventType()) {
				functionDef := decls.Functions[subscriptionDef.Function]
				log.Debug("Now triggering event to ", subscriptionDef.Function)
				_, log, err := sb.Invoke(wsEvent, functionDef.Code, decls.Environment)
				if err != nil {
					app.logCallback(subscriptionDef.Function, err.Error())
				}
				if log != "" {
					app.logCallback(subscriptionDef.Function, log)
				}
			}
		}
	}
}
