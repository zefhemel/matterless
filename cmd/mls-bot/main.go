package main

import (
	"github.com/zefhemel/matterless/pkg/bot"
	"os"
	"strconv"
	"time"

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

	apiBindPort, err := strconv.Atoi(os.Getenv("api_bind_port"))
	if err != nil {
		log.Fatal("Could not parse $api_bind_port: ", err)
	}

	mb, err := bot.NewBot(os.Getenv("server"), os.Getenv("token"), apiBindPort)
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
