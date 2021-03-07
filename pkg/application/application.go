package application

import (
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/declaration"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/util"
)

type Application struct {
	declarations *declaration.Declarations
	eventSources map[string]eventsource.EventSource

	// Callbacks
	// TODO: Consider switching to channels instead?
	logCallback func(kind string, message string)
}

func (app *Application) SetDeclarations(decls *declaration.Declarations) error {
	app.declarations = decls

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

func NewApplication(logCallback func(kind, message string)) *Application {
	return &Application{
		eventSources: map[string]eventsource.EventSource{},
		logCallback:  logCallback,
	}
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
