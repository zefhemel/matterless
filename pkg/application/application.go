package application

import (
	"github.com/mattermost/mattermost-server/model"
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
	code        string
	definitions *definition.Definitions

	// Runtime
	eventSources map[string]eventsource.EventSource
	sandbox      *sandbox.DockerSandbox
	adminClient  *model.Client4

	// Callbacks
	// TODO: Consider switching to channels instead?
	logCallback func(kind string, message string)
}

func NewApplication(adminClient *model.Client4, logCallback func(kind, message string)) *Application {
	return &Application{
		adminClient:  adminClient,
		eventSources: map[string]eventsource.EventSource{},
		logCallback:  logCallback,
		// TODO: Make this configurable
		sandbox: sandbox.NewDockerSandbox(30*time.Second, 1*time.Minute),
	}
}

func (app *Application) handleFunctionCall(name definition.FunctionID, event interface{}) interface{} {
	functionDef := app.definitions.Functions[name]
	log.Debug("Now triggering event to ", name)
	result, log, err := app.sandbox.Invoke(event, app.definitions.CompileFunctionCode(functionDef.Code), app.definitions.Environment)
	if err != nil {
		app.logCallback(string(name), err.Error())
	}
	if log != "" {
		app.logCallback(string(name), log)
	}
	return result
}

func (app *Application) CurrentCode() string {
	return app.code
}

func (app *Application) Eval(code string) error {
	log.Debug("Parsing and checking definitions...")
	app.code = code
	decls, err := definition.Parse(code)
	if err != nil {
		return err
	}
	decls.Normalize()

	app.definitions = decls

	results := definition.Check(decls)
	if results.String() != "" {
		// Error while checking
		return errors.Wrap(errors.New(results.String()), "declaration check")
	}

	log.Debug("Starting listeners...")
	app.Stop()

	// Rebuild the whole thing
	for name, def := range decls.MattermostClients {
		mmSource, err := eventsource.NewMatterMostSource(name, def, app.handleFunctionCall)
		if err != nil {
			return err
		}
		app.eventSources[name] = mmSource
		mmSource.Start()
		mmSource.ExposeEnvironment(decls.Environment)
		log.Debug("Starting Mattermost client: ", name)
	}

	for name, def := range decls.Bots {
		botSource, err := eventsource.NewBotSource(app.adminClient, name, def, app.handleFunctionCall)
		if err != nil {
			return err
		}
		app.eventSources[name] = botSource
		botSource.Start()
		botSource.ExposeEnvironment(decls.Environment)
		log.Debug("Starting Bot: ", name)
	}

	for name, def := range decls.APIGateways {
		apiG := eventsource.NewAPIGatewaySource(def, app.handleFunctionCall)
		if err != nil {
			return err
		}
		app.eventSources[name] = apiG
		log.Debug("Starting API Gateway: ", name)
		apiG.Start()
	}

	for name, def := range decls.SlashCommands {
		scmd := eventsource.NewSlashCommandSource(app.adminClient, def, app.handleFunctionCall)
		if err != nil {
			return err
		}
		app.eventSources[name] = scmd
		log.Debug("Starting Slashcommand: ", name)
		scmd.Start()
	}

	log.Debug("Testing functions...")
	testResults := checker.TestDeclarations(decls, app.sandbox)
	for name, functionResult := range testResults.Functions {
		if functionResult.Logs != "" {
			app.logCallback(string(name), functionResult.Logs)
		}
	}
	if testResults.String() != "" {
		return errors.Wrap(errors.New(testResults.String()), "test run")
	}
	log.Debug("All ready to go!")
	return nil
}

func (app *Application) Stop() {
	// First, stop all event sources
	for sourceName, eventSource := range app.eventSources {
		log.Debug("Stopping ", sourceName)
		eventSource.Stop()
	}
	// Then stop all functions in sandbox
	log.Debug("Stopping sandbox")
	app.sandbox.FlushAll()
}
