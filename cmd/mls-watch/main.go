package main

import (
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var serverURL string
var rootToken string

func updateApp(appName, code string) {
	fmt.Printf("Updating app %s...\n", appName)
	client := http.DefaultClient

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", serverURL, appName), strings.NewReader(code))
	if err != nil {
		fmt.Println("Updating app fail: ", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", rootToken))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Updating app fail: ", err)
		return
	}

	bodyData, _ := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	fmt.Printf("All good!\n\n%s\n", bodyData)
}

func deleteApp(appName string) {
	client := http.DefaultClient

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", serverURL, appName), nil)
	if err != nil {
		fmt.Println("Deleting app fail: ", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", rootToken))
	if _, err := client.Do(req); err != nil {
		fmt.Println("Deleting app fail: ", err)
	}
}

func main() {
	flag.StringVar(&serverURL, "url", "http://localhost:8222", "URL to the Matterless server")
	flag.StringVar(&rootToken, "token", "", "Root token for the matterless server")
	flag.Parse()

	// File watch the definition file and reload on changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	go watch(watcher)

	for _, path := range flag.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Could not open file %s: %s", path, err)
			continue
		}

		updateApp(filepath.Base(path), string(data))
		err = watcher.Add(path)
		if err != nil {
			panic(err)
		}
	}

	// Handle Ctrl-c gracefully
	killing := make(chan os.Signal)
	signal.Notify(killing, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-killing
		watcher.Close()
		for _, path := range flag.Args() {
			appName := filepath.Base(path)
			fmt.Printf("Disabling app %s...\n", appName)
			deleteApp(appName)
		}
		os.Exit(0)
	}()

	for {
		time.Sleep(time.Minute)
	}
}

func watch(watcher *fsnotify.Watcher) {
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
				updateApp(filepath.Base(path), string(data))
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}
