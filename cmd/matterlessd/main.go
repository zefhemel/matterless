package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/application"
	config "github.com/zefhemel/matterless/pkg/config"
	"time"
)

func main() {
	log.SetLevel(log.DebugLevel)

	cfg := config.FromEnv()

	appContainer := application.NewContainer(cfg)
	if err := appContainer.Start(); err != nil {
		log.Fatal("Could not start app container", err)
	}
	appContainer.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) (interface{}, error) {
		if le, ok := eventData.(application.LogEntry); ok {
			if le.LogEntry.Instance == nil {
				return nil, nil
			}

			log.Infof("[App: %s | Function: %s] %s", le.AppName, le.LogEntry.Instance.Name(), le.LogEntry.Message)
		} else {
			log.Error("Received log event that's not an application.LogEntry ", eventData)
		}
		return nil, nil
	})

	// Idle loop, everything runs in go-routines
	for {
		time.Sleep(30 * time.Second)
	}
}
