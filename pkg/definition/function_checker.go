package definition

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zefhemel/matterless/pkg/sandbox"
)

type TestResults struct {
	Functions map[FunctionID]error
}

func TestDeclarations(defs *Definitions, sandbox sandbox.Sandbox) TestResults {
	testResults := TestResults{
		Functions: map[FunctionID]error{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for name, def := range defs.Functions {
		_, err := sandbox.Function(ctx, string(name), defs.Environment, defs.ModulesForLanguage(def.Language), def.Code)
		testResults.Functions[name] = err
	}
	return testResults
}

func (tr *TestResults) String() string {
	errorMessageParts := make([]string, 0, 10)
	for functionName, functionResult := range tr.Functions {
		if functionResult != nil {
			errorMessageParts = append(errorMessageParts, fmt.Sprintf("[Function: %s Error] %s", functionName, functionResult.Error()))
		}
	}
	return strings.Join(errorMessageParts, "\n")
}
