package main

import (
	"github.com/zefhemel/matterless/pkg/bot"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}

	mb, err := bot.NewBot(os.Getenv("server"), os.Getenv("token"))
	if err != nil {
		log.Printf("Error connecting: %+v \n", err)
		return
	}

	log.Println("Connecting...")
	err = mb.Start()
	if err != nil {
		log.Error(err)
	}
}
