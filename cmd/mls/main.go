package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/client"
	"github.com/zefhemel/matterless/pkg/config"
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
	cmd.Flags().StringVarP(&cfg.NatsUrl, "nats", "n", "nats://localhost:4222", "NATS server to connect to")

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

			runConsole(mlsClient, args)
		},
	}

	cmdDeploy.Flags().BoolVarP(&watch, "watch", "w", false, "watch apps for changes and redeploy")
	cmdDeploy.Flags().StringVar(&url, "url", "", "URL or Matterless server to deploy to")
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
	cmd.Flags().StringVar(&url, "url", "", "URL or Matterless server to deploy to")
	cmd.Flags().StringVarP(&adminToken, "token", "t", "", "Root token for Matterless server")

	return cmd
}

func rootCommand() *cobra.Command {
	cfg := config.NewConfig()

	var cmd = &cobra.Command{
		Use:   "mls",
		Short: "Run Matterless in server mode",
		Run: func(cmd *cobra.Command, args []string) {
			runServer(cfg)
			busyLoop()
		},
	}
	cmd.Flags().IntVarP(&cfg.APIBindPort, "port", "p", 8222, "Port to listen to")
	cmd.Flags().StringVarP(&cfg.AdminToken, "token", "t", "", "Admin API token")
	cmd.Flags().StringVar(&cfg.DataDir, "data", "./mls-data", "location to keep Matterless state")
	cmd.Flags().StringVarP(&cfg.NatsUrl, "nats", "n", "nats://localhost:4222", "NATS server to connect to")

	return cmd
}

func main() {
	log.SetLevel(log.DebugLevel)

	cmd := rootCommand()
	cmd.AddCommand(runCommand(), deployCommand(), attachCommand(), ppCommand())
	cmd.Execute()
}

func runServer(cfg *config.Config) *application.Container {
	appContainer, err := application.NewContainer(cfg)
	if err != nil {
		log.Fatal("Could not start app container", err)
	}

	// Subscribe to all logs and write to stdout
	appContainer.ClusterConnection().Subscribe(fmt.Sprintf("%s.*.function.*.log", cfg.NatsPrefix), func(m *nats.Msg) {
		parts := strings.Split(m.Subject, ".") // mls.myapp.function.MyFunction.log
		log.Infof("[%s | %s] %s", parts[1], parts[3], string(m.Data))
	})

	// Handle Ctrl-c gracefully
	killing := make(chan os.Signal)
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
