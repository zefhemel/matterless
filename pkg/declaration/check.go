package declaration

import (
	"errors"
	"fmt"
	"strings"
)

type CheckResults struct {
	Functions     map[string][]error
	Sources       map[string][]error
	Subscriptions map[string][]error
}

func (cr *CheckResults) String() string {
	errorMessageParts := make([]string, 0, 10)
	for functionName, functionErrors := range cr.Functions {
		errorMessageParts = collectPrettyErrors(functionErrors, errorMessageParts, functionName, "Function")
	}
	for sourceName, sourceErrors := range cr.Sources {
		errorMessageParts = collectPrettyErrors(sourceErrors, errorMessageParts, sourceName, "Source")
	}
	for subscriptionName, subscriptionErrors := range cr.Subscriptions {
		errorMessageParts = collectPrettyErrors(subscriptionErrors, errorMessageParts, subscriptionName, "Subscription")
	}
	return strings.Join(errorMessageParts, "\n")
}

func collectPrettyErrors(errorList []error, errorMessageParts []string, name string, kind string) []string {
	for _, err := range errorList {
		errorMessageParts = append(errorMessageParts, fmt.Sprintf("[%s: %s] %s", kind, name, err))
	}
	return errorMessageParts
}

func Check(declarations *Declarations) CheckResults {
	return CheckResults{
		Functions:     checkFunctions(declarations),
		Sources:       checkSources(declarations),
		Subscriptions: checkSubscriptions(declarations),
	}
}

func checkSubscriptions(declarations *Declarations) map[string][]error {
	subscriptionResults := make(map[string][]error)
	for subscriptionName, subscriptionDef := range declarations.Subscriptions {
		errorList := make([]error, 0, 5)
		if subscriptionDef.Source == "" {
			errorList = append(errorList, errors.New("no Source specified"))
		} else if _, found := declarations.Sources[subscriptionDef.Source]; !found {
			errorList = append(errorList, fmt.Errorf("subscription source %s not found", subscriptionDef.Source))
		}
		if subscriptionDef.Function == "" {
			errorList = append(errorList, errors.New("no Function to trigger specified"))
		} else if _, found := declarations.Functions[subscriptionDef.Function]; !found {
			errorList = append(errorList, fmt.Errorf("function %s not found", subscriptionDef.Function))
		}
		subscriptionResults[subscriptionName] = errorList
	}
	return subscriptionResults
}

func checkSources(declarations *Declarations) map[string][]error {
	sourceResults := make(map[string][]error)
	for sourceName, sourceDef := range declarations.Sources {
		errorList := make([]error, 0, 5)
		if sourceDef.Type != "Mattermost" {
			errorList = append(errorList, errors.New("only support event type is Mattermost"))
		}
		if sourceDef.URL == "" {
			errorList = append(errorList, errors.New("no URL specified"))
		}
		if sourceDef.Token == "" {
			errorList = append(errorList, errors.New("no Token specified"))
		}
		sourceResults[sourceName] = errorList
	}
	return sourceResults
}

func checkFunctions(declarations *Declarations) map[string][]error {
	functionResults := make(map[string][]error)
	for functionName, functionDef := range declarations.Functions {
		errorList := make([]error, 0, 5)
		if functionName == "" {
			errorList = append(errorList, errors.New("Empty function name"))
		}
		if functionDef.Code == "" {
			errorList = append(errorList, errors.New("Empty function body"))
		}
		functionResults[functionName] = errorList
	}
	return functionResults
}
