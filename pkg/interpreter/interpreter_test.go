package interpreter_test

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/declaration"
	"github.com/zefhemel/matterless/pkg/interpreter"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

//go:embed test/test1.md
var test1Md string

func TestInterpreter(t *testing.T) {
	defs, err := declaration.Parse([]string{test1Md})
	assert.NoError(t, err)
	sb := sandbox.NewMockSandbox(nil, "Ok", nil)
	results := interpreter.TestDeclarations(defs, sb)
	assert.Equal(t, nil, results.FunctionResults["TestFunction1"].Error)
}

func TestNodeInterpreter(t *testing.T) {
	defs, err := declaration.Parse([]string{test1Md})
	assert.NoError(t, err)
	sb := sandbox.NewNodeSandbox("node")
	results := interpreter.TestDeclarations(defs, sb)
	assert.NoError(t, results.FunctionResults["TestFunction1"].Error)
	assert.Equal(t, "Hello world!", results.FunctionResults["TestFunction1"].Logs)
	assert.True(t, results.FunctionResults["TestFunction1"].Result.(bool))
	assert.Equal(t, "Hello world 2!", results.FunctionResults["TestFunction2"].Logs)
}
