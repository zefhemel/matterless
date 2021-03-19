package definition

import (
	"errors"
	"fmt"
	"github.com/robfig/cron"
	"strings"
)

type CheckResults struct {
	Functions         map[FunctionID][]error
	MattermostClients map[string][]error
	APIs              []error
	Crons             []error
	SlashCommands     map[string][]error
	Bots              map[string][]error
	Libraries         map[string][]error
}

func (cr *CheckResults) String() string {
	errorMessageParts := make([]string, 0, 10)
	for functionName, functionErrors := range cr.Functions {
		errorMessageParts = collectPrettyErrors(functionErrors, errorMessageParts, string(functionName), "Function")
	}
	for name, errs := range cr.MattermostClients {
		errorMessageParts = collectPrettyErrors(errs, errorMessageParts, name, "MattermostClient")
	}
	errorMessageParts = collectPrettyErrors(cr.APIs, errorMessageParts, "", "API")
	for name, errs := range cr.SlashCommands {
		errorMessageParts = collectPrettyErrors(errs, errorMessageParts, name, "SlashCommand")
	}
	for name, errs := range cr.Bots {
		errorMessageParts = collectPrettyErrors(errs, errorMessageParts, name, "Bot")
	}
	errorMessageParts = collectPrettyErrors(cr.Crons, errorMessageParts, "", "Cron")
	for libraryName, libraryErrors := range cr.Libraries {
		errorMessageParts = collectPrettyErrors(libraryErrors, errorMessageParts, libraryName, "Library")
	}
	return strings.Join(errorMessageParts, "\n")
}

func collectPrettyErrors(errorList []error, errorMessageParts []string, name string, kind string) []string {
	for _, err := range errorList {
		errorMessageParts = append(errorMessageParts, fmt.Sprintf("[%s: %s] %s", kind, name, err))
	}
	return errorMessageParts
}

func Check(declarations *Definitions) CheckResults {
	return CheckResults{
		Functions:         checkFunctions(declarations),
		MattermostClients: checkMattermostClients(declarations),
		APIs:              checkAPIs(declarations),
		SlashCommands:     checkSlashCommands(declarations),
		Bots:              checkBots(declarations),
		Crons:             checkCrons(declarations),
		Libraries:         checkLibraries(declarations),
	}
}

func checkMattermostClients(declarations *Definitions) map[string][]error {
	results := make(map[string][]error)
	for name, def := range declarations.MattermostClients {
		errorList := make([]error, 0, 5)
		if def.URL == "" {
			errorList = append(errorList, errors.New("no 'url' specified"))
		}
		if def.Token == "" {
			errorList = append(errorList, errors.New("no 'token' specified"))
		}
		for _, functionIDs := range def.Events {
			for _, functionID := range functionIDs {
				if !declarations.FunctionExists(functionID) {
					errorList = append(errorList, fmt.Errorf("function %s not found", functionID))
				}
			}
		}
		results[name] = errorList
	}
	return results
}

func checkAPIs(declarations *Definitions) []error {
	errorList := make([]error, 0, 5)
	for _, endpointDef := range declarations.APIs {
		if endpointDef.Path == "" {
			errorList = append(errorList, errors.New("no 'path' defined for endpoint"))
		}
		if endpointDef.Function == "" {
			errorList = append(errorList, errors.New("no 'function' defined for endpoint"))
		} else if !declarations.FunctionExists(endpointDef.Function) {
			errorList = append(errorList, fmt.Errorf("function %s not found", endpointDef.Function))
		}
	}
	return errorList
}

func checkSlashCommands(declarations *Definitions) map[string][]error {
	results := make(map[string][]error)
	for name, def := range declarations.SlashCommands {
		errorList := make([]error, 0, 5)
		if def.TeamName == "" {
			errorList = append(errorList, errors.New("no 'team_name' specified"))
		}
		if def.Trigger == "" {
			errorList = append(errorList, errors.New("no 'trigger' specified"))
		}
		if !declarations.FunctionExists(def.Function) {
			errorList = append(errorList, fmt.Errorf("function %s not found", def.Function))
		}
		results[name] = errorList
	}
	return results
}

func checkBots(declarations *Definitions) map[string][]error {
	results := make(map[string][]error)
	for name, def := range declarations.Bots {
		errorList := make([]error, 0, 5)
		if def.Username == "" {
			errorList = append(errorList, errors.New("no 'username' specified"))
		}
		for _, functionIDs := range def.Events {
			for _, functionID := range functionIDs {
				if !declarations.FunctionExists(functionID) {
					errorList = append(errorList, fmt.Errorf("function %s not found", functionID))
				}
			}
		}
		results[name] = errorList
	}
	return results
}

func checkCrons(declarations *Definitions) []error {
	errorList := make([]error, 0, 5)
	for _, def := range declarations.Crons {
		if def.Schedule == "" {
			errorList = append(errorList, errors.New("no 'schedule' specified"))
		}

		if _, err := cron.Parse(def.Schedule); err != nil {
			errorList = append(errorList, fmt.Errorf("invalid 'schedule' format: %s", err))
		}
		if !declarations.FunctionExists(def.Function) {
			errorList = append(errorList, fmt.Errorf("function %s not found", def.Function))
		}
	}
	return errorList
}

func checkFunctions(declarations *Definitions) map[FunctionID][]error {
	results := make(map[FunctionID][]error)
	for name, def := range declarations.Functions {
		errorList := make([]error, 0, 5)
		if name == "" {
			errorList = append(errorList, errors.New("Empty function name"))
		}
		if def.Code == "" {
			errorList = append(errorList, errors.New("Empty function body"))
		}
		results[name] = errorList
	}
	return results
}

func checkLibraries(declarations *Definitions) map[string][]error {
	results := make(map[string][]error)
	for name, def := range declarations.Libraries {
		errorList := make([]error, 0, 5)
		if def.Code == "" {
			errorList = append(errorList, errors.New("Empty library body"))
		}
		results[name] = errorList
	}
	return results
}
