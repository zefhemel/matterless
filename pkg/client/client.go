package client

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type MatterlessClient struct {
	URL   string
	Token string
}

func NewMatterlessClient(url string, token string) *MatterlessClient {
	return &MatterlessClient{
		URL:   url,
		Token: token,
	}
}

func (client *MatterlessClient) Deploy(files []string, watch bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Could not open file %s: %s", path, err)
			continue
		}
		appName := filepath.Base(path)
		fmt.Printf("Deploying %s\n", appName)
		client.updateApp(appName, string(data))
		err = watcher.Add(path)
		if err != nil {
			panic(err)
		}
	}

	if watch {
		// File watch the definition file and reload on changes
		client.watcher(watcher)
	}
}

func (client *MatterlessClient) Delete(files []string) {
	for _, path := range files {
		appName := filepath.Base(path)
		fmt.Printf("Undeploying %s\n", appName)
		client.deleteApp(appName)
	}
}

func (client *MatterlessClient) updateApp(appName, code string) {
	fmt.Printf("Updating app %s...\n", appName)

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", client.URL, appName), strings.NewReader(code))
	if err != nil {
		fmt.Println("Updating app fail: ", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	resp, err := http.DefaultClient.Do(req)
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

func (client *MatterlessClient) deleteApp(appName string) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", client.URL, appName), nil)
	if err != nil {
		fmt.Println("Deleting app fail: ", err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", client.Token))
	if _, err := http.DefaultClient.Do(req); err != nil {
		fmt.Println("Deleting app fail: ", err)
	}
}

func (client *MatterlessClient) watcher(watcher *fsnotify.Watcher) {
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
				client.updateApp(filepath.Base(path), string(data))
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}
