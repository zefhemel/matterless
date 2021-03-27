package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	log.SetLevel(log.DebugLevel)
	cfg := config.FromEnv()

	var (
		watch    bool
		bindPort int
		url      string
	)
	var cmdRun = &cobra.Command{
		Use:   "run [file1.md] ...",
		Short: "Run Matterless and run listed apps",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.APIBindPort = bindPort
			cfg.RootToken = util.TokenGenerator()
			cfg.APIExternalURL = url
			go runServer(cfg)
			client := NewMatterlessClient(cfg.APIExternalURL, cfg.RootToken)
			client.Deploy(args, watch)
			for {
				time.Sleep(1 * time.Minute)
			}
		},
	}
	cmdRun.Flags().BoolVarP(&watch, "watch", "w", false, "watch apps for changes and reload")
	cmdRun.Flags().IntVarP(&bindPort, "port", "p", 8222, "port to bind to")
	cmdRun.Flags().StringVarP(&url, "url", "u", "http://localhost:8222", "URL to externally available matterless server")

	var (
		token string
	)

	var cmdDeploy = &cobra.Command{
		Use:   "deploy [file1.md] ..",
		Short: "Deploy applications to a server",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Deploy: " + strings.Join(args, " "))
		},
	}

	cmdDeploy.Flags().BoolVarP(&watch, "watch", "w", false, "watch apps for changes and redeploy")
	cmdDeploy.Flags().IntVarP(&bindPort, "port", "p", 8222, "port to bind to")
	cmdDeploy.Flags().StringVarP(&url, "url", "u", "http://localhost:8222", "Matterless server URL to deploy to")
	cmdDeploy.Flags().StringVarP(&token, "token", "t", "", "Token to use for authentication")

	var rootCmd = &cobra.Command{
		Use:   "mls",
		Short: "Matterless is friction-free serverless",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Matterless server mode")
			runServer(cfg)

			// Idle loop, everything runs in go-routines
			for {
				time.Sleep(30 * time.Second)
			}

		},
	}
	rootCmd.AddCommand(cmdRun, cmdDeploy)
	rootCmd.Execute()

}

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

func runServer(cfg config.Config) {
	appContainer, err := application.NewContainer(cfg)
	if err != nil {
		log.Fatal("Could not start app container", err)
	}
	appContainer.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) (interface{}, error) {
		if le, ok := eventData.(application.LogEntry); ok {
			if le.LogEntry.Instance == nil {
				return nil, nil
			}

			log.Infof("[App: %s | Function: %s] %s", le.AppName, le.LogEntry.Instance.Name(), le.LogEntry.Message)
		} else {
			log.Error("Received log event that's not an application.LogEntry ", eventData)
		}
		return nil, nil
	})

	// Handle Ctrl-c gracefully
	killing := make(chan os.Signal)
	signal.Notify(killing, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-killing
		appContainer.Close()
		os.Exit(0)
	}()
}
