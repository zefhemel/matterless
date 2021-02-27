package interpreter

import (
	"github.com/zefhemel/matterless/pkg/declaration"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

type TestResults struct {
	FunctionResults map[string]FunctionTestResult
}

type FunctionTestResult struct {
	Error  error
	Logs   string
	Result interface{}
}

func TestDeclarations(defs declaration.Declarations, sandbox sandbox.Sandbox) TestResults {
	testResults := TestResults{
		FunctionResults: map[string]FunctionTestResult{},
	}
	for funcName, funcDef := range defs.Functions {
		ftr := FunctionTestResult{}
		ftr.Result, ftr.Logs, ftr.Error = sandbox.Invoke(struct{}{}, funcDef.Code, map[string]string{})
		testResults.FunctionResults[funcName] = ftr
	}
	return testResults
}
