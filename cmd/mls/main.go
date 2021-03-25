package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/application"
	config "github.com/zefhemel/matterless/pkg/config"
	"os"
	"os/signal"
	"path/filepath"
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

	if len(os.Args) == 1 {
		log.Fatal("Usage: mls [application1.md] [application2.md] ...")
	}

	adminClient := model.NewAPIv4Client(cfg.MattermostURL)
	adminClient.SetOAuthToken(cfg.AdminToken)

	appContainer := application.NewContainer(cfg.APIBindPort)
	if err := appContainer.Start(); err != nil {
		log.Fatal("Could not start app container", err)
	}
	// File watch the definition file and reload on changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	go watch(watcher, appContainer)

	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Could not open file %s: %s", path, err)
		}

		appName := filepath.Base(path)

		app := application.NewApplication(cfg, adminClient, appName)

		go func() {
			for le := range app.Logs() {
				if le.Instance == nil {
					continue
				}
				log.Infof("[Function %s] %s", le.Instance.Name(), le.Message)
			}
		}()

		err = app.Eval(string(data))
		if err != nil {
			log.Fatal(err)
		}
		appContainer.Register(appName, app)
		err = watcher.Add(path)
		if err != nil {
			log.Fatal(err)
		}

		log.Infof("Application %s successfully loaded, here is a summary of what is live:", path)
		log.Info(app.Definitions().Markdown())
	}

	// Handle Ctrl-c gracefully
	killing := make(chan os.Signal)
	signal.Notify(killing, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-killing
		appContainer.Close()
		os.Exit(0)
	}()

	// Idle loop, everything runs in go-routines
	for {
		time.Sleep(30 * time.Second)
	}
}

func watch(watcher *fsnotify.Watcher, appContainer *application.Container) {
eventLoop:
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				path := event.Name
				log.Infof("Definition %s modified, reloading...", path)
				data, err := os.ReadFile(path)
				if err != nil {
					log.Fatalf("Could not open file %s: %s", path, err)
					continue eventLoop
				}
				app := appContainer.Get(filepath.Base(path))
				err = app.Eval(string(data))
				if err != nil {
					log.Errorf("Error processing %s: %s", path, err)
					continue eventLoop
				}
				log.Infof("Application %s successfully loaded, here is a summary of what is live:", path)
				log.Info(app.Definitions().Markdown())
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}
