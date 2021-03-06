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
	stopped  chan struct{}
}

func NewMatterMostSource(url, token string) (*MatterMostSource, error) {
	wsURL := strings.Replace(url, "http:", "ws:", 1)
	wsURL = strings.Replace(wsURL, "https:", "wss:", 1)
	log.Debug("Websocket URL: ", wsURL)
	mms := &MatterMostSource{
		stopping: make(chan struct{}),
		stopped:  make(chan struct{}),
	}

	var err *model.AppError
	mms.wsClient, err = model.NewWebSocketClient4(wsURL, token)
	if err != nil {
		log.Error("Connecting to websocket", err)
		return nil, err
	}

	return mms, nil
}

// Events filtered by SubscribedEvents
func (mms *MatterMostSource) Events() chan interface{} {
	return mms.events
}

func stringSliceContains(stringSlice []string, needle string) bool {
	for _, s := range stringSlice {
		if needle == s {
			return true
		}
	}
	return false
}

func (mms *MatterMostSource) Start() error {
	err := mms.wsClient.Connect()

	if err != nil {
		return err
	}

	mms.wsClient.Listen()

	// Initialize event channel
	mms.events = make(chan interface{})

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
