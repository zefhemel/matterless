package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/application"
	config "github.com/zefhemel/matterless/pkg/config"
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

	cfg := config.FromEnv()

	filename := "matterless.md"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not open file %s: %s", filename, err)
	}

	adminClient := model.NewAPIv4Client(cfg.MattermostURL)
	adminClient.SetOAuthToken(cfg.AdminToken)

	appContainer := application.NewContainer(cfg.APIBindPort)

	defaultApp := application.NewApplication(cfg, adminClient, "default", func(kind, message string) {
		log.Infof("%s: %s", kind, message)
	})
	appContainer.Register("default", defaultApp)

	if err := appContainer.Start(); err != nil {
		log.Fatal("Could not start app container", err)
	}

	err = defaultApp.Eval(string(data))
	if err != nil {
		log.Fatal(err)
	}

	// Handle Ctrl-c gracefully
	killing := make(chan os.Signal)
	signal.Notify(killing, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-killing
		defaultApp.Close()
		appContainer.Stop()
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
					err = defaultApp.Eval(string(data))
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
