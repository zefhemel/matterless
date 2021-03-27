package application

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

type LogEntry struct {
	AppName  string
	LogEntry sandbox.LogEntry
}

type Container struct {
	eventBus   eventbus.EventBus
	apps       map[string]*Application
	apiGateway *APIGateway
}

func NewContainer(config config.Config) (*Container, error) {
	appMap := map[string]*Application{}
	c := &Container{
		apps:     appMap,
		eventBus: eventbus.NewLocalEventBus(),
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
	app.EventBus().Subscribe("logs:*", func(eventName string, eventData interface{}) (interface{}, error) {
		if le, ok := eventData.(sandbox.LogEntry); ok {
			c.EventBus().Publish(fmt.Sprintf("logs:%s:%s", name, le.Instance.Name()), LogEntry{
				AppName:  name,
				LogEntry: le,
			})
		} else {
			log.Fatal("Did not get sandbox.LogEntry", eventData)
		}
		return nil, nil
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
