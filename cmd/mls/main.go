package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/client"
	"github.com/zefhemel/matterless/pkg/config"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	log.SetLevel(log.DebugLevel)

	// Linux docker parent host
	apiHost := "172.17.0.1"
	if runtime.GOOS != "linux" {
		apiHost = "host.docker.internal"
	}

	runWatch := false
	var runConfig config.Config
	var cmdRun = &cobra.Command{
		Use:   "run [file1.md] ...",
		Short: "Run Matterless and run listed apps",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runConfig.APIURL = fmt.Sprintf("http://%s:%d", apiHost, runConfig.APIBindPort)
			runServer(runConfig, false)
			mlsClient := client.NewMatterlessClient(runConfig.APIURL, runConfig.RootToken)
			mlsClient.Deploy(args, runWatch)
			busyLoop()
		},
	}
	cmdRun.Flags().BoolVarP(&runWatch, "watch", "w", false, "watch apps for changes and reload")
	cmdRun.Flags().IntVarP(&runConfig.APIBindPort, "port", "p", 8222, "Port to bind API Gateway to")
	cmdRun.Flags().StringVarP(&runConfig.RootToken, "token", "t", "", "Root API token")
	cmdRun.Flags().StringVar(&runConfig.DataDir, "data", "", "Path to keep Matterless state")

	var (
		deployWatch bool
		deployURL   string
		deployToken string
	)
	var cmdDeploy = &cobra.Command{
		Use:   "deploy [file1.md] ..",
		Short: "Deploy applications to a server",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if deployURL == "" {
				fmt.Println("Did not provide Matterless URL to connect to via --url")
				os.Exit(1)
			}
			if deployToken == "" {
				fmt.Println("Did not provide Matterless admin token via --token")
				os.Exit(1)
			}
			mlsClient := client.NewMatterlessClient(deployURL, deployToken)
			mlsClient.Deploy(args, deployWatch)
		},
	}

	cmdDeploy.Flags().BoolVarP(&deployWatch, "watch", "w", false, "watch apps for changes and redeploy")
	cmdDeploy.Flags().StringVar(&deployURL, "url", "", "URL or Matterless server to deploy to")
	cmdDeploy.Flags().StringVarP(&deployToken, "token", "t", "", "Root token for Matterless server")

	var serverConfig config.Config

	var rootCmd = &cobra.Command{
		Use:   "mls",
		Short: "Matterless is friction-free serverless",
		Run: func(cmd *cobra.Command, args []string) {
			serverConfig.APIURL = fmt.Sprintf("http://%s:%d", apiHost, serverConfig.APIBindPort)
			runServer(serverConfig, true)
			busyLoop()
		},
	}
	rootCmd.Flags().IntVarP(&serverConfig.APIBindPort, "port", "p", 8222, "Port to bind API Gateway to")
	rootCmd.Flags().StringVarP(&serverConfig.RootToken, "token", "t", "", "Root API token")
	rootCmd.Flags().StringVar(&serverConfig.DataDir, "data", "./mls-data", "location to keep Matterless state")

	rootCmd.AddCommand(cmdRun, cmdDeploy)
	rootCmd.Execute()

}

func runServer(cfg config.Config, loadApps bool) {
	appContainer, err := application.NewContainer(cfg)
	if err != nil {
		log.Fatal("Could not start app container", err)
	}
	appContainer.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) {
		if le, ok := eventData.(application.LogEntry); ok {
			if le.LogEntry.Instance == nil {
				return
			}

			log.Infof("[App: %s | Function: %s] %s", le.AppName, le.LogEntry.Instance.Name(), le.LogEntry.Message)
		} else {
			log.Error("Received log event that's not an application.LogEntry ", eventData)
		}
	})

	if loadApps {
		if err := appContainer.LoadAppsFromDisk(); err != nil {
			log.Errorf("Could not load apps from disk: %s", err)
		}
	}

	// Handle Ctrl-c gracefully
	killing := make(chan os.Signal)
	signal.Notify(killing, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-killing
		appContainer.Close()
		os.Exit(0)
	}()
}

func busyLoop() {
	for {
		time.Sleep(1 * time.Minute)
	}
}
