package eventsource

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/definition"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
)

type MatterMostSource struct {
	clientName         string
	wsClient           *model.WebSocketClient
	token              string
	def                *definition.MattermostClientDef
	stopping           chan struct{}
	stopped            chan struct{}
	functionInvokeFunc definition.FunctionInvokeFunc
}

func NewMatterMostSource(clientName string, def *definition.MattermostClientDef, functionInvokeFunc definition.FunctionInvokeFunc) (*MatterMostSource, error) {
	wsURL := strings.Replace(def.URL, "http:", "ws:", 1)
	wsURL = strings.Replace(wsURL, "https:", "wss:", 1)
	log.Debug("Websocket URL: ", wsURL)
	mms := &MatterMostSource{
		clientName:         clientName,
		stopping:           make(chan struct{}),
		stopped:            make(chan struct{}),
		def:                def,
		functionInvokeFunc: functionInvokeFunc,
	}

	var err *model.AppError
	mms.wsClient, err = model.NewWebSocketClient4(wsURL, def.Token)
	if err != nil {
		log.Error("Connecting to websocket", err)
		return nil, err
	}

	// start listening
	if err := mms.start(); err != nil {
		return nil, err
	}

	return mms, nil
}

func (mms *MatterMostSource) start() error {
	err := mms.wsClient.Connect()

	if err != nil {
		return err
	}

	mms.wsClient.Listen()

	go func() {
	eventLoop:
		for {
			select {
			case evt, ok := <-mms.wsClient.EventChannel:
				if ok {
					if eventListeners, ok := mms.def.Events[evt.Event]; ok {
						for _, eventListener := range eventListeners {
							go mms.functionInvokeFunc(eventListener, evt)
						}
					} else if eventListeners, ok := mms.def.Events["all"]; ok {
						for _, eventListener := range eventListeners {
							go mms.functionInvokeFunc(eventListener, evt)
						}
					}
				} else {
					// Channel closed, likely due to socket disconnect. Reconnect
				reconnectLoop:
					for {
						log.Info("Disconnected.")
						time.Sleep(2 * time.Second)
						log.Info("Reconnecting...")
						err := mms.start()
						if err != nil {
							log.Error(err)
						} else {
							break reconnectLoop
						}
					}
				}
			case <-mms.stopping:
				break eventLoop
			}
		}
		mms.wsClient.Close()
		mms.stopped <- struct{}{}
	}()
	return nil
}

func (mms *MatterMostSource) ExtendDefinitions(defs *definition.Definitions) {
	// clientName == empty is the matterless bot, let's not expose those credentials
	if mms.clientName != "" {
		defs.Environment[fmt.Sprintf("%s_URL", strings.ToUpper(mms.clientName))] = mms.def.URL
		defs.Environment[fmt.Sprintf("%s_TOKEN", strings.ToUpper(mms.clientName))] = mms.def.Token
	}
}

func (mms *MatterMostSource) Close() {
	mms.stopping <- struct{}{}
	// Wait for the connection to actually close
	<-mms.stopped
}

var _ EventSource = &MatterMostSource{}
