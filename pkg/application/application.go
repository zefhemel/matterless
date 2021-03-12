package application

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/checker"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"time"
)

type Application struct {
	// Definitions
	declarations *definition.Definitions

	// Runtime
	eventSources map[string]eventsource.EventSource
	sandbox      *sandbox.DockerSandbox

	// Callbacks
	// TODO: Consider switching to channels instead?
	logCallback func(kind string, message string)
}

func NewApplication(logCallback func(kind, message string)) *Application {
	return &Application{
		eventSources: map[string]eventsource.EventSource{},
		logCallback:  logCallback,
		// TODO: Make this configurable
		sandbox: sandbox.NewDockerSandbox(30*time.Second, 1*time.Minute),
	}
}

func (app *Application) Eval(code string) error {
	log.Debug("Parsing and checking definitions...")
	decls, err := definition.Parse(code)
	if err != nil {
		return err
	}
	decls.Normalize()

	results := definition.Check(decls)
	if results.String() != "" {
		// Error while checking
		return errors.Wrap(errors.New(results.String()), "declaration check")
	}
	log.Debug("Testing functions...")
	testResults := checker.TestDeclarations(decls, app.sandbox)
	for _, functionResult := range testResults.Functions {
		if functionResult.Logs != "" {
			app.logCallback("Function test", functionResult.Logs)
		}
	}
	if testResults.String() != "" {
		return errors.Wrap(errors.New(testResults.String()), "test run")
	}

	log.Debug("Starting listeners...")
	// First, stop all event sources
	for sourceName, eventSource := range app.eventSources {
		log.Debug("Stopping listener: ", sourceName)
		eventSource.Stop()
	}

	// Rebuild the whole thing
	for name, def := range decls.MattermostClients {
		mmSource, err := eventsource.NewMatterMostSource(def, func(name definition.FunctionID, event interface{}) interface{} {
			functionDef := decls.Functions[name]
			log.Debug("Now triggering event to ", name)
			_, log, err := app.sandbox.Invoke(event, decls.CompileFunctionCode(functionDef.Code), decls.Environment)
			if err != nil {
				app.logCallback(string(name), err.Error())
			}
			if log != "" {
				app.logCallback(string(name), log)
			}
			return nil
		})
		if err != nil {
			return err
		}
		app.eventSources[name] = mmSource
		mmSource.Start()
		log.Debug("Starting Mattermost client: ", name)
	}

	for name, def := range decls.APIGateways {
		apiG := eventsource.NewAPIGatewaySource(def, func(name definition.FunctionID, event interface{}) interface{} {
			functionDef := decls.Functions[name]
			log.Debug("Now triggering event to ", name)
			result, log, err := app.sandbox.Invoke(event, decls.CompileFunctionCode(functionDef.Code), decls.Environment)
			if err != nil {
				app.logCallback(string(name), err.Error())
			}
			if log != "" {
				app.logCallback(string(name), log)
			}
			return result
		})
		if err != nil {
			return err
		}
		app.eventSources[name] = apiG
		apiG.Start()
		log.Debug("Starting Mattermost client: ", name)
	}

	return nil
}
