package main

import (
	"bufio"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/zefhemel/matterless/pkg/definition"
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
	var (
		watch  bool
		attach bool
	)
	cfg := config.NewConfig()
	var cmd = &cobra.Command{
		Use:   "run [file.md]",
		Short: "Run matterless in ad-hoc mode for specified markdown definition file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.LoadApps = false
			// Spin up own nats server
			cfg.ClusterNatsUrl = fmt.Sprintf("nats://localhost:%d", util.FindFreePort(4222))
			container := runServer(cfg)
			defer container.Close()

			apiURL := fmt.Sprintf("http://localhost:%d", cfg.APIBindPort)
			mlsClient := client.NewMatterlessClient(apiURL, cfg.AdminToken)
			appPath := args[0]
			loadApp(appPath, mlsClient)
			if watch {
				watcher(appPath, func() {
					loadApp(appPath, mlsClient)
				})
			}
			if attach {
				runConsole(mlsClient, appPath, func() {
					loadApp(appPath, mlsClient)
				}, func() {
					container.Close()
					os.Exit(0)
				})
			}
			if !attach {
				// To make sure we don't exit
				busyLoop()
			}
		},
	}
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch apps for changes and reload")
	cmd.Flags().BoolVarP(&attach, "attach", "a", false, "attach to console")
	cmd.Flags().IntVarP(&cfg.APIBindPort, "port", "p", 8222, "Port to bind API Gateway to")
	cmd.Flags().StringVarP(&cfg.AdminToken, "token", "t", "", "Admin API token")
	cmd.Flags().StringVar(&cfg.DataDir, "data", "./mls-data", "Path to keep Matterless state")
	cmd.Flags().StringVarP(&cfg.ClusterNatsUrl, "nats", "n", "nats://localhost:4222", "NATS server to connect to")

	return cmd
}

func loadApp(appPath string, mlsClient *client.MatterlessClient) {
	appName := client.AppNameFromPath(appPath)
	code, err := os.ReadFile(appPath)
	if err != nil {
		log.Fatalf("Could not read file: %s", err)
	}
	for {
		defs, err := definition.Check(appPath, string(code), "")
		if err != nil {
			log.Fatal(err)
		}
		if err := mlsClient.DeployApp(appName, defs.Markdown()); err != nil {
			if missingConfigsErr, ok := err.(*client.ConfigIssuesError); ok {
				askForConfigs(mlsClient, appName, defs.Config, missingConfigsErr.ConfigIssues)
				continue
			}
			fmt.Printf("Failed to deploy: %s\n", err)
			return
		}
		break
	}
}

func askForConfigs(mlsClient *client.MatterlessClient, appName string, configSpec map[string]*definition.TypeSchema, issues map[string]string) error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Some configuration is required for %s, please enter values (in YAML):\n", appName)
	for configName, issueError := range issues {
		fmt.Printf("Config option '%s': %s\n", configName, issueError)
		if configSpec[configName].Description != "" {
			fmt.Printf("Description: %s\n", configSpec[configName].Description)
		}
		fmt.Print("Value: ")
		if scanner.Scan() {
			line := scanner.Text()
			yamlValue, err := util.YamlUnmarshal(string(line))
			if err != nil {
				fmt.Printf("[ERROR] Invalid config value: %s\n", err)
				continue
			}
			if err := mlsClient.StorePut(appName, configName, yamlValue); err != nil {
				fmt.Printf("[ERROR] Storing config failed: %s\n", err)
				continue
			}
		}
	}
	return nil
}

func deployCommand() *cobra.Command {
	var (
		attach     bool
		watch      bool
		url        string
		adminToken string
	)
	var cmd = &cobra.Command{
		Use:   "deploy [file1.md] ..",
		Short: "Deploy apps to a matterless server",
		Args:  cobra.ExactArgs(1),
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
			appPath := args[0]
			loadApp(appPath, mlsClient)

			if watch {
				watcher(appPath, func() {
					loadApp(appPath, mlsClient)
				})
			}
			if attach {
				runConsole(mlsClient, appPath, func() {
					loadApp(appPath, mlsClient)
				}, func() {
					os.Exit(0)
				})
			}
			if watch && !attach {
				// To make sure we don't exit
				busyLoop()
			}
		},
	}
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch apps for changes and reload")
	cmd.Flags().BoolVarP(&attach, "attach", "a", false, "attach to console")
	cmd.Flags().StringVarP(&url, "url", "u", "http://localhost:8222", "URL or Matterless server to deploy to")
	cmd.Flags().StringVarP(&adminToken, "token", "t", "", "Root token for Matterless server")

	return cmd
}

func watcher(filePath string, reloadCallback func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Starting watcher: %s", err)
	}
	watcher.Add(filePath)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					reloadCallback()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Error(err)
				}
			}
		}
	}()
}

func receiveLogs(mlsClient *client.MatterlessClient, appName string) {
	ch, err := mlsClient.EventStream(appName)
	if err != nil {
		log.Fatalf("Could not open stream: %s", err)
	}

	go func() {
		for message := range ch {
			data, ok := message.Data.(map[string]interface{})
			if ok {
				fmt.Printf("[LOG] %s: %s", data["function"], data["message"])
			}
		}
	}()
	if err := mlsClient.SubscribeEvent("*.log"); err != nil {
		log.Fatalf("Could not subscribe: %s", err)
	}
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
			runConsole(mlsClient, "", func() {
				fmt.Println("No app loaded, nothing to reload")
			}, func() {
				os.Exit(0)
			})
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
	appContainer.ClusterEventBus().SubscribeContainerLogs(func(appName, funcName, message string) {
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
