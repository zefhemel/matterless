package main

import (
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/declaration"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

type userState struct {
	declarations declaration.Declarations
	eventSources map[string]eventsource.EventSource
}

func (mb *MatterlessBot) evalDeclarations(userID string, decls declaration.Declarations) error {
	userS, ok := mb.userState[userID]
	if !ok {
		userS = &userState{
			eventSources: map[string]eventsource.EventSource{},
		}
	}

	userS.declarations = decls

	// First, stop all event sources
	for sourceName, eventSource := range userS.eventSources {
		log.Debug("Stopping listener: ", sourceName)
		eventSource.Stop()
	}

	// Rebuild the whole thing
	for sourceName, sourceDef := range decls.Sources {
		mmSource, err := eventsource.NewMatterMostSource(sourceDef.URL, sourceDef.Token)
		if err != nil {
			return err
		}
		userS.eventSources[sourceName] = mmSource
		mmSource.Start()
		log.Debug("Starting listener: ", sourceName)
		go mb.eventProcessor(userID, mmSource, decls)
	}

	mb.userState[userID] = userS
	return nil
}

func (mb *MatterlessBot) eventProcessor(userID string, source eventsource.EventSource, decls declaration.Declarations) {
	sb := sandbox.NewNodeDockerSandbox()
	env := map[string]string{}
	for evt := range source.Events() {
		wsEvent, ok := evt.(*model.WebSocketEvent)
		if !ok {
			log.Debug("Got non websocket event", evt)
			continue
		}

		log.Debug("Got event to process ", wsEvent)

		for _, subscriptionDef := range decls.Subscriptions {
			if stringSliceContains(subscriptionDef.EventTypes, wsEvent.EventType()) {
				functionDef := decls.Functions[subscriptionDef.Function]
				invokeEnv := env
				if subscriptionDef.PassSourceCredentials {
					sourceDef := decls.Sources[subscriptionDef.Source]
					invokeEnv = map[string]string{
						"SOURCE_URL":   sourceDef.URL,
						"SOURCE_TOKEN": sourceDef.Token,
					}
				}
				log.Debug("Now triggering event to ", subscriptionDef.Function)
				_, log, err := sb.Invoke(wsEvent, functionDef.Code, invokeEnv)
				if err != nil {
					mb.postFunctionLog(userID, subscriptionDef.Function, err.Error())
				}
				if log != "" {
					mb.postFunctionLog(userID, subscriptionDef.Function, log)
				}
			}
		}
	}
}
