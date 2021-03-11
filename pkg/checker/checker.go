package checker

import (
	"fmt"
	"strings"

	"github.com/zefhemel/matterless/pkg/declaration"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

type TestResults struct {
	Functions map[string]FunctionTestResult
}

type FunctionTestResult struct {
	Error  error
	Logs   string
	Result interface{}
}

func TestDeclarations(defs *declaration.Declarations, sandbox sandbox.Sandbox) TestResults {
	testResults := TestResults{
		Functions: map[string]FunctionTestResult{},
	}

	for name, def := range defs.Functions {
		ftr := FunctionTestResult{}
		ftr.Result, ftr.Logs, ftr.Error = sandbox.Invoke(struct{}{}, defs.CompileFunctionCode(def.Code), defs.Environment)
		testResults.Functions[name] = ftr
	}
	return testResults
}

func (tr *TestResults) String() string {
	errorMessageParts := make([]string, 0, 10)
	for functionName, functionResult := range tr.Functions {
		if functionResult.Error != nil {
			errorMessageParts = append(errorMessageParts, fmt.Sprintf("[Function: %s Error] %s", functionName, functionResult.Error.Error()))
			// errorMessageParts = append(errorMessageParts, fmt.Sprintf("[Function: %s Logs] %s", functionName, functionResult.Logs))
		}
	}
	return strings.Join(errorMessageParts, "\n")
}
