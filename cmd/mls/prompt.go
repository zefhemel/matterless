package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/client"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
)

func completer(in prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "r", Description: "Reload app from disk"},
		{Text: "restart", Description: "restart application"},
		{Text: "use", Description: "[appName] - switch to app context"},
		{Text: "delete-app", Description: "[appName] - delete app and its data"},
		{Text: "list", Description: "list all running apps"},
		{Text: "put", Description: "[key] [value] — puts a value in the store"},
		{Text: "get", Description: "[key] — retrieve a value from the store"},
		{Text: "del", Description: "[key] — delete a value from the store"},
		{Text: "query-prefix", Description: "[key-prefix] — query keys from the store, return with values"},
		{Text: "keys", Description: "[key-prefix] — query keys from the store"},
		{Text: "trigger", Description: "[eventName] [evenData] — trigger an event"},
		{Text: "invoke", Description: "[functionName] [evenData] — invoke a function"},
		{Text: "exit", Description: "Exit"},
	}
	w := in.GetWordBeforeCursor()
	if strings.HasPrefix(in.Text, "use ") || strings.HasPrefix(in.Text, "delete-app ") {
		appNameSuggestions := make([]prompt.Suggest, len(promptContext.allAppNames))
		for i, appName := range promptContext.allAppNames {
			appNameSuggestions[i] = prompt.Suggest{
				Text:        appName,
				Description: "",
			}
		}
		return prompt.FilterHasPrefix(appNameSuggestions, w, false)
	}
	if strings.HasPrefix(in.Text, "invoke ") && promptContext.defs != nil {
		functionNameSuggestions := make([]prompt.Suggest, 0)
		for fnName, _ := range promptContext.defs.Functions {
			functionNameSuggestions = append(functionNameSuggestions, prompt.Suggest{
				Text:        string(fnName),
				Description: "",
			})
		}
		return prompt.FilterHasPrefix(functionNameSuggestions, w, false)
	}
	if strings.HasPrefix(in.Text, "trigger ") && promptContext.defs != nil {
		eventNameSuggestions := make([]prompt.Suggest, 0)
		for fnName, _ := range promptContext.defs.Events {
			eventNameSuggestions = append(eventNameSuggestions, prompt.Suggest{
				Text:        fnName,
				Description: "",
			})
		}
		return prompt.FilterHasPrefix(eventNameSuggestions, w, false)
	}
	if w == "" {
		return []prompt.Suggest{}
	}
	return prompt.FilterHasPrefix(s, w, false)
}

func executor(cmd string) {
	blocks := strings.Split(cmd, " ")
	if len(blocks) == 0 {
		return
	}
	switch blocks[0] {
	case "reload", "r":
		promptContext.reloadCallback()
	case "exit":
		fmt.Println("Bye!")
		promptContext.exitCallback()
	case "use":
		if len(blocks) != 2 {
			fmt.Println("You should specify an app name")
			return
		}
		if _, err := promptContext.client.GetAppCode(blocks[1]); err != nil {
			fmt.Println("App does not exist")
			return
		}
		promptContext.appName = blocks[1]
		fetchMetadata()
	case "list":
		appNames, err := promptContext.client.ListApps()
		if err != nil {
			fmt.Printf("Error fetching app names: %s\n", err)
			return
		}
		for _, name := range appNames {
			fmt.Printf("- %s\n", name)
		}
	case "delete-app":
		if len(blocks) != 2 {
			fmt.Println("You should specify an app name")
			return
		}
		if err := promptContext.client.DeleteApp(blocks[1]); err != nil {
			fmt.Println("App could not be deleted ", err.Error())
			return
		}
		promptContext.appName = ""
	case "get":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		key := blocks[1]
		if val, err := promptContext.client.StoreGet(promptContext.appName, key); err != nil {
			fmt.Printf("Failed to retrieve from datastore: %s\n", err)
		} else {
			fmt.Println(util.MustJsonString(val))
		}
	case "query-prefix":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		prefix := ""
		if len(blocks) == 2 {
			prefix = blocks[1]
		}
		if val, err := promptContext.client.StoreQueryPrefix(promptContext.appName, prefix); err != nil {
			fmt.Printf("Failed to retrieve from datastore: %s\n", err)
		} else {
			for _, result := range val {
				fmt.Printf("- Key: %s Value: %s\n", result[0], util.MustJsonString(result[1]))
			}
		}
	case "keys":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		prefix := ""
		if len(blocks) == 2 {
			prefix = blocks[1]
		}
		if val, err := promptContext.client.StoreQueryPrefix(promptContext.appName, prefix); err != nil {
			fmt.Printf("Failed to retrieve from datastore: %s\n", err)
		} else {
			for _, result := range val {
				fmt.Printf("%s\n", result[0])
			}
		}
	case "del":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		key := blocks[1]
		if err := promptContext.client.StoreDel(promptContext.appName, key); err != nil {
			fmt.Printf("Failed to delete from datastore: %s\n", err)
		} else {
			fmt.Println("OK")
		}
	case "put":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		key := blocks[1]
		valJson := strings.Join(blocks[2:], " ")
		var obj interface{}
		if err := json.Unmarshal([]byte(valJson), &obj); err != nil {
			fmt.Printf("Could not parse value as JSON (%s): %s\n", err, valJson)
			return
		}
		if err := promptContext.client.StorePut(promptContext.appName, key, obj); err != nil {
			fmt.Printf("Failed to put into datastore: %s\n", err)
		} else {
			fmt.Println("Done")
		}
	case "restart":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		if err := promptContext.client.RestartApp(promptContext.appName); err != nil {
			fmt.Printf("Failed to restart: %s\n", err)
		} else {
			fmt.Println("Done.")
		}
	case "trigger":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		eventName := blocks[1]
		valJson := strings.Join(blocks[2:], " ")
		if valJson == "" {
			valJson = "{}"
		}

		var obj interface{}
		if err := json.Unmarshal([]byte(valJson), &obj); err != nil {
			fmt.Printf("Could not parse value as JSON (%s): %s\n", err, valJson)
			return
		}
		if err := promptContext.client.TriggerEvent(promptContext.appName, eventName, obj); err != nil {
			fmt.Printf("Failed to trigger event: %s\n", err)
		} else {
			fmt.Println("Done.")
		}
	case "invoke":
		if promptContext.appName == "" {
			fmt.Println("Please select an app first with 'use appname'")
			return
		}
		functionName := blocks[1]
		valJson := strings.Join(blocks[2:], " ")
		if valJson == "" {
			valJson = "{}"
		}
		var obj interface{}
		if err := json.Unmarshal([]byte(valJson), &obj); err != nil {
			fmt.Printf("Could not parse value as JSON (%s): %s\n", err, valJson)
			return
		}
		if result, err := promptContext.client.InvokeFunction(promptContext.appName, functionName, obj); err != nil {
			fmt.Printf("Failed to invoke function: %s\n", err)
		} else {
			fmt.Println(util.MustJsonString(result))
		}
	}
}

type PromptContext struct {
	appName        string
	client         *client.MatterlessClient
	allAppNames    []string
	defs           *definition.Definitions
	reloadCallback func()
	exitCallback   func()
}

var promptContext = &PromptContext{}

func livePrefix() (string, bool) {
	if promptContext.appName == "" {
		return "", false
	}
	return promptContext.appName + "> ", true
}

func runConsole(c *client.MatterlessClient, filePath string, reloadCallback func(), exitCallback func()) {
	promptContext.client = c
	promptContext.reloadCallback = reloadCallback
	promptContext.exitCallback = exitCallback
	if filePath != "" {
		promptContext.appName = client.AppNameFromPath(filePath)
	}
	go metaDataFetcher()
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
		prompt.OptionLivePrefix(livePrefix),
		prompt.OptionTitle("mls"),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn: func(buffer *prompt.Buffer) {
				exitCallback()
			},
		}),
	)
	p.Run()
}

// Fetches meta data (app names, definitions) regularly for auto complete
func metaDataFetcher() {
	time.Sleep(time.Second)
	for {
		fetchMetadata()
		time.Sleep(10 * time.Second)
	}
}

func fetchMetadata() {
	allApps, err := promptContext.client.ListApps()
	if err != nil {
		log.Errorf("Could not fetch apps: %s", err)
	} else {
		promptContext.allAppNames = allApps
	}
	if promptContext.appName != "" {
		defs, err := promptContext.client.GetDefinitions(promptContext.appName)
		if err != nil {
			log.Errorf("Could not fetch meta data for %s: %s", promptContext.appName, err)
		} else {
			promptContext.defs = defs
		}
	}
}
