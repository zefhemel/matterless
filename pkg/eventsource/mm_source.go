package eventsource

import (
	"github.com/zefhemel/matterless/pkg/definition"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
)

type MatterMostSource struct {
	wsClient           *model.WebSocketClient
	token              string
	def                *definition.MattermostClientDef
	stopping           chan struct{}
	stopped            chan struct{}
	functionInvokeFunc FunctionInvokeFunc
}

type FunctionInvokeFunc func(name definition.FunctionID, event interface{}) interface{}

func NewMatterMostSource(def *definition.MattermostClientDef, functionInvokeFunc FunctionInvokeFunc) (*MatterMostSource, error) {
	wsURL := strings.Replace(def.URL, "http:", "ws:", 1)
	wsURL = strings.Replace(wsURL, "https:", "wss:", 1)
	log.Debug("Websocket URL: ", wsURL)
	mms := &MatterMostSource{
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

	return mms, nil
}

func (mms *MatterMostSource) Start() error {
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
							mms.functionInvokeFunc(eventListener, evt)
						}
					} else if eventListeners, ok := mms.def.Events["all"]; ok {
						for _, eventListener := range eventListeners {
							mms.functionInvokeFunc(eventListener, evt)
						}
					}
				} else {
					// Channel closed, likely due to socket disconnect. Reconnect
				reconnectLoop:
					for {
						log.Info("Disconnected.")
						time.Sleep(2 * time.Second)
						log.Info("Reconnecting...")
						err := mms.Start()
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

func (mms *MatterMostSource) Stop() {
	mms.stopping <- struct{}{}
	// Wait for the connection to actually close
	<-mms.stopped
}

var _ EventSource = &MatterMostSource{}
