package application

import (
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

type LogEntry struct {
	AppName  string
	LogEntry sandbox.LogEntry
}

type Container struct {
	apps       map[string]*Application
	apiGateway *APIGateway
	logChannel chan LogEntry
}

func NewContainer(config config.Config) *Container {
	appMap := map[string]*Application{}
	c := &Container{
		apps:       appMap,
		logChannel: make(chan LogEntry),
	}
	c.apiGateway = NewAPIGateway(config, c, func(appName string, name definition.FunctionID, event interface{}) interface{} {
		return appMap[appName].InvokeFunction(name, event)
	})

	return c
}

func (c *Container) Start() error {
	return c.apiGateway.Start()
}

func (c *Container) Close() {
	for _, app := range c.apps {
		app.Close()
	}
	c.apiGateway.Stop()
	close(c.logChannel)
}

func (c *Container) Register(name string, app *Application) {
	c.apps[name] = app
	go func() {
		for le := range app.Logs() {
			c.logChannel <- LogEntry{name, le}
		}
	}()
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

func (c *Container) Logs() chan LogEntry {
	return c.logChannel
}
