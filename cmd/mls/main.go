package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/client"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/util"
)

func runCommand() *cobra.Command {
	watch := false
	cfg := config.NewConfig()
	var cmd = &cobra.Command{
		Use:   "run [file1.md] ...",
		Short: "Run matterless in ad-hoc mode for specified markdown definition files",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.LoadApps = false
			// Spin up own nats server
			cfg.ClusterNatsUrl = fmt.Sprintf("nats://localhost:%d", util.FindFreePort(4222))
			container := runServer(cfg)
			defer container.Close()

			apiURL := fmt.Sprintf("http://localhost:%d", cfg.APIBindPort)
			mlsClient := client.NewMatterlessClient(apiURL, cfg.AdminToken)
			if err := mlsClient.DeployAppFiles(args, watch); err != nil {
				fmt.Printf("Failed to deploy: %s\n", err)
				return
			}

			runConsole(mlsClient, args)
		},
	}
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch apps for changes and reload")
	cmd.Flags().IntVarP(&cfg.APIBindPort, "port", "p", 8222, "Port to bind API Gateway to")
	cmd.Flags().StringVarP(&cfg.AdminToken, "token", "t", "", "Admin API token")
	cmd.Flags().StringVar(&cfg.DataDir, "data", "./mls-data", "Path to keep Matterless state")
	cmd.Flags().StringVarP(&cfg.ClusterNatsUrl, "nats", "n", "nats://localhost:4222", "NATS server to connect to")

	return cmd
}

func deployCommand() *cobra.Command {
	var (
		watch      bool
		url        string
		adminToken string
	)
	var cmdDeploy = &cobra.Command{
		Use:   "deploy [file1.md] ..",
		Short: "Deploy apps to a matterless server",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if url == "" {
				fmt.Println("Did not provide Matterless URL to connect to via --url")
				os.Exit(1)
			}
			if adminToken == "" {
				fmt.Println("Did not provide Matterless admin token via --token")
				os.Exit(1)
			}

			mlsClient := client.NewMatterlessClient(url, adminToken)
			if err := mlsClient.DeployAppFiles(args, watch); err != nil {
				fmt.Printf("Failed to deploy: %s\n", err)
				return
			}

			if watch {
				busyLoop()
			}
			//runConsole(mlsClient, args)
		},
	}

	cmdDeploy.Flags().BoolVarP(&watch, "watch", "w", false, "watch apps for changes and redeploy")
	cmdDeploy.Flags().StringVar(&url, "url", "http://localhost:8222", "URL or Matterless server to deploy to")
	cmdDeploy.Flags().StringVarP(&adminToken, "token", "t", "", "Root token for Matterless server")

	return cmdDeploy
}

func attachCommand() *cobra.Command {
	var (
		url        string
		adminToken string
	)
	var cmd = &cobra.Command{
		Use:   "attach",
		Short: "Attach to a running matterless server console",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			mlsClient := client.NewMatterlessClient(url, adminToken)
			runConsole(mlsClient, []string{})
		},
	}
	cmd.Flags().StringVar(&url, "url", "http://localhost:8222", "URL of matterless server to connect to")
	cmd.Flags().StringVarP(&adminToken, "token", "t", "", "Root token for Matterless server")

	return cmd
}

func infoCommand() *cobra.Command {
	var (
		url        string
		adminToken string
	)
	var cmd = &cobra.Command{
		Use:   "info",
		Short: "Retrieve cluster information",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			mlsClient := client.NewMatterlessClient(url, adminToken)
			info, err := mlsClient.ClusterInfo()
			if err != nil {
				fmt.Printf("Error fetching cluster info: %s\n", err)
				return
			}
			fmt.Println(util.MustJsonString(info))
		},
	}
	cmd.Flags().StringVar(&url, "url", "http://localhost:8222", "URL of matterless server to connect to")
	cmd.Flags().StringVarP(&adminToken, "token", "t", "", "Root token for Matterless server")

	return cmd
}

func rootCommand() *cobra.Command {
	cfg := config.NewConfig()

	var cmd = &cobra.Command{
		Use:   "mls",
		Short: "Run Matterless in server mode",
		Run: func(cmd *cobra.Command, args []string) {
			container := runServer(cfg)
			defer container.Close()
			busyLoop()
		},
	}
	cmd.Flags().IntVarP(&cfg.APIBindPort, "port", "p", 8222, "Port to listen to")
	cmd.Flags().StringVarP(&cfg.AdminToken, "token", "t", "", "Admin API token")
	cmd.Flags().StringVar(&cfg.DataDir, "data", "./mls-data", "location to keep Matterless state")
	cmd.Flags().StringVarP(&cfg.ClusterNatsUrl, "nats", "n", "nats://localhost:4222", "NATS server to connect to")

	return cmd
}

func main() {
	log.SetLevel(log.DebugLevel)

	cmd := rootCommand()
	cmd.AddCommand(runCommand(), deployCommand(), attachCommand(), infoCommand(), ppCommand())
	cmd.Execute()
}

func runServer(cfg *config.Config) *application.Container {
	appContainer, err := application.NewContainer(cfg)
	if err != nil {
		log.Fatal("Could not start app container", err)
	}

	// Subscribe to all logs and write to stdout
	appContainer.ClusterEventBus().SubscribeContainerLogs("*.function.*.log", func(appName, funcName, message string) {
		log.Infof("LOG [%s | %s]: %s", appName, funcName, message)
	})

	if err := appContainer.Start(); err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	// Handle Ctrl-c gracefully
	killing := make(chan os.Signal, 1)
	signal.Notify(killing, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-killing
		log.Info("Shutting down...")
		appContainer.Close()
		os.Exit(0)
	}()

	return appContainer
}

func busyLoop() {
	for {
		time.Sleep(1 * time.Minute)
	}
}
