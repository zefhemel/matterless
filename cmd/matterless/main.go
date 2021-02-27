package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.Println("Hello world!")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}

	mb, err := NewBot(os.Getenv("server"), os.Getenv("token"))
	if err != nil {
		fmt.Printf("Error connecting: %+v \n", err)
		return
	}

	fmt.Println("Connecting...")
	err = mb.Start()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

}
