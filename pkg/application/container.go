package application

import (
	"github.com/zefhemel/matterless/pkg/definition"
)

type Container struct {
	apps       map[string]*Application
	apiGateway *APIGateway
}

func NewContainer(apiGatewayBindPort int) *Container {
	appMap := map[string]*Application{}
	return &Container{
		apps: appMap,
		apiGateway: NewAPIGateway(apiGatewayBindPort, appMap, func(appName string, name definition.FunctionID, event interface{}) interface{} {
			return appMap[appName].InvokeFunction(name, event)
		}),
	}
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
