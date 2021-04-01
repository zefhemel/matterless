package application

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/util"
	"os"
	"strings"
)

type LogEntry struct {
	AppName  string
	LogEntry sandbox.LogEntry
}

type Container struct {
	eventBus   eventbus.EventBus
	apps       map[string]*Application
	apiGateway *APIGateway
	config     *config.Config
}

func NewContainer(config *config.Config) (*Container, error) {
	appMap := map[string]*Application{}
	c := &Container{
		apps:     appMap,
		config:   config,
		eventBus: eventbus.NewLocalEventBus(),
	}

	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, errors.Wrap(err, "create data dir")
	}

	if config.AdminToken == "" {
		tokenFilePath := fmt.Sprintf("%s/root.token", config.DataDir)
		if buf, err := os.ReadFile(tokenFilePath); err != nil {
			config.AdminToken = util.TokenGenerator()
			if err := os.WriteFile(tokenFilePath, []byte(config.AdminToken), 0600); err != nil {
				log.Errorf("Could not write to %s: %s", tokenFilePath, err)
			}
		} else {
			config.AdminToken = string(buf)
		}
	}

	c.apiGateway = NewAPIGateway(config, c, func(appName string, name definition.FunctionID, event interface{}) interface{} {
		return appMap[appName].InvokeFunction(name, event)
	})
	if err := c.apiGateway.Start(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Container) EventBus() eventbus.EventBus {
	return c.eventBus
}

func (c *Container) Close() {
	for _, app := range c.apps {
		app.Close()
	}
	c.apiGateway.Stop()
}

func (c *Container) Register(name string, app *Application) {
	c.apps[name] = app

	// Listen to sandbox log events, republish on parent eventbus
	app.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) {
		if le, ok := eventData.(sandbox.LogEntry); ok {
			if le.Instance == nil {
				return
			}
			c.EventBus().Publish(fmt.Sprintf("logs:%s:%s", name, le.Instance.Name()), LogEntry{
				AppName:  name,
				LogEntry: le,
			})
		} else {
			log.Fatal("Did not get sandbox.LogEntry", eventData)
		}
	})
}

func (c *Container) Get(name string) *Application {
	return c.apps[name]
}

func (c *Container) UnRegister(name string) {
	if app, ok := c.apps[name]; ok {
		app.Close()
		delete(c.apps, name)
	}
}

func (c *Container) LoadAppsFromDisk() error {
	files, err := os.ReadDir(c.config.DataDir)
	if err != nil {
		return err
	}
fileLoop:
	for _, file := range files {
		appName := file.Name()
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			if _, err := os.Stat(fmt.Sprintf("%s/%s/application.md", c.config.DataDir, appName)); err != nil {
				// No application file
				continue fileLoop
			}
			// It's an app, let's load it
			app, err := NewApplication(c.config, appName)
			if err != nil {
				return errors.Wrapf(err, "creating app: %s", appName)
			}
			c.Register(appName, app)
			if err := app.LoadFromDisk(); err != nil {
				return errors.Wrapf(err, "loading app: %s", appName)
			}
		}
	}
	return nil
}

func (c *Container) Config() *config.Config {
	return c.config
}
