package interpreter

import (
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

type TestResults struct {
	FunctionResults map[string]FunctionTestResult
}

type FunctionTestResult struct {
	Error  error
	Logs   []string
	Result interface{}
}

func TestDefinitions(defs definition.Definitions, sandbox sandbox.Sandbox) TestResults {
	testResults := TestResults{
		FunctionResults: map[string]FunctionTestResult{},
	}
	for funcName, funcDef := range defs.Functions {
		ftr := FunctionTestResult{}
		ftr.Result, ftr.Logs, ftr.Error = sandbox.Invoke(map[string]string{
			"type": "test",
		}, funcDef.Code, map[string]string{})
		testResults.FunctionResults[funcName] = ftr
	}
	return testResults
}
