package application

import (
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

func NewContainer(config config.Config) *Container {
	appMap := map[string]*Application{}
	c := &Container{
		apps:     appMap,
		eventBus: eventbus.NewLocalEventBus(),
	}
	c.apiGateway = NewAPIGateway(config, c, func(appName string, name definition.FunctionID, event interface{}) interface{} {
		return appMap[appName].InvokeFunction(name, event)
	})

	return c
}

func (c *Container) EventBus() eventbus.EventBus {
	return c.eventBus
}

func (c *Container) Start() error {
	return c.apiGateway.Start()
}

func (c *Container) Close() {
	for _, app := range c.apps {
		app.Close()
	}
	c.apiGateway.Stop()
}

func (c *Container) Register(name string, app *Application) {
	c.apps[name] = app
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
