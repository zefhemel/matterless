package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Hello world!")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}

	mb, err := NewBot(os.Getenv("server"), os.Getenv("wsserver"), os.Getenv("token"))
	if err != nil {
		fmt.Printf("Error connecting: %+v \n", err)
		return
	}

	for {
		fmt.Println("Connecting...")
		err = mb.Listen()
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		time.Sleep(5 * time.Second)
	}

}
