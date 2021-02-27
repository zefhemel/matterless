package eventsource

import (
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
)

type MatterMostSource struct {
	wsClient *model.WebSocketClient
	token    string
	events   chan interface{}
	stopping chan struct{}
}

func NewMatterMostSource(url, token string) (*MatterMostSource, error) {
	wsURL := strings.Replace(url, "http:", "ws:", 1)
	wsURL = strings.Replace(wsURL, "https:", "wss:", 1)
	log.Debug("Websocket URL: ", wsURL)
	mms := &MatterMostSource{
		events:   make(chan interface{}),
		stopping: make(chan struct{}),
	}

	var err *model.AppError
	mms.wsClient, err = model.NewWebSocketClient4(wsURL, token)
	if err != nil {
		log.Error("Connecting to websocket", err)
		return nil, err
	}

	return mms, nil
}

func (mms *MatterMostSource) Events() chan interface{} {
	return mms.events
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
					mms.events <- evt
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
		close(mms.events)
	}()
	return nil
}

func (mms *MatterMostSource) Stop() {
	mms.stopping <- struct{}{}
}

var _ EventSource = &MatterMostSource{}
