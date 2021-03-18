package main

import (
	"github.com/zefhemel/matterless/pkg/bot"
	"github.com/zefhemel/matterless/pkg/config"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)

	cfg := config.FromEnv()
	mb, err := bot.NewBot(cfg)
	if err != nil {
		log.Printf("Error connecting: %+v \n", err)
		return
	}

	log.Println("Connecting...")
	err = mb.Start()
	if err != nil {
		log.Error(err)
	}

	for {
		time.Sleep(30 * time.Second)
	}
}
