package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/application"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Depends on the following environment variables
// server: URL to MM server (admin account)
// token: token for admin account
// external_host: externally accessible hostname to access this machine

func main() {
	log.SetLevel(log.DebugLevel)

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}

	filename := "matterless.md"
	if len(os.Args) > 0 {
		filename = os.Args[1]
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not open file %s: %s", filename, err)
	}

	adminClient := model.NewAPIv4Client(os.Getenv("server"))
	adminClient.SetOAuthToken(os.Getenv("token"))

	app := application.NewApplication(adminClient, func(kind, message string) {
		log.Infof("%s: %s", kind, message)
	})
	err = app.Eval(string(data))
	if err != nil {
		log.Fatal(err)
	}

	// Handle Ctr-c gracefully
	killing := make(chan os.Signal)
	signal.Notify(killing, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-killing
		app.Stop()
		os.Exit(0)
	}()

	// File watch the definition file and reload on changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
	eventLoop:
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Infof("Definition %s  modified, reloading...", filename)
					data, err := os.ReadFile(filename)
					if err != nil {
						log.Fatalf("Could not open file %s: %s", filename, err)
						continue eventLoop
					}
					err = app.Eval(string(data))
					if err != nil {
						log.Errorf("Error processing %s: %s", filename, err)
						continue eventLoop
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(filename)
	if err != nil {
		log.Fatal(err)
	}

	for {
		time.Sleep(30 * time.Second)
	}
}
