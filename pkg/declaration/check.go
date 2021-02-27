package declaration

import (
	"errors"
	"fmt"
	"strings"
)

type CheckResults struct {
	Functions map[string]FunctionCheckResult
}

func (cr *CheckResults) String() string {
	errorMessageParts := make([]string, 0, 10)
	for functionName, functionResult := range cr.Functions {
		for _, err := range functionResult.Errors {
			errorMessageParts = append(errorMessageParts, fmt.Sprintf("[Function: %s] %s", functionName, err))
		}
	}
	return strings.Join(errorMessageParts, "\n")
}

type FunctionCheckResult struct {
	Errors []error
}

func Check(declarations Declarations) CheckResults {
	return CheckResults{
		Functions: checkFunctions(declarations),
	}
}

func checkFunctions(declarations Declarations) map[string]FunctionCheckResult {
	functionResults := make(map[string]FunctionCheckResult)
	for functionName, functionDef := range declarations.Functions {
		errorList := make([]error, 0, 5)
		if functionName == "" {
			errorList = append(errorList, errors.New("Empty function name"))
		}
		if functionDef.Code == "" {
			errorList = append(errorList, errors.New("Empty function body"))
		}
		functionResults[functionName] = FunctionCheckResult{
			Errors: errorList,
		}
	}
	return functionResults
}
