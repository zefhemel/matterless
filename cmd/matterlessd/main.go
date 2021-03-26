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
	go func() {
		for le := range appContainer.Logs() {
			if le.LogEntry.Instance == nil {
				continue
			}
			log.Infof("[App: %s | Function: %s] %s", le.AppName, le.LogEntry.Instance.Name(), le.LogEntry.Message)
		}
	}()

	// Idle loop, everything runs in go-routines
	for {
		time.Sleep(30 * time.Second)
	}
}
