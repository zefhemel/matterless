package interpreter_test

import (
	_ "embed"
	"strings"
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
	assert.Equal(t, nil, results.Functions["TestFunction1"].Error)
}

func TestNodeInterpreter(t *testing.T) {
	defs, err := declaration.Parse([]string{test1Md})
	assert.NoError(t, err)
	sb := sandbox.NewNodeSandbox("node")
	results := interpreter.TestDeclarations(defs, sb)
	assert.NoError(t, results.Functions["TestFunction1"].Error)
	assert.Equal(t, "Hello world!", results.Functions["TestFunction1"].Logs)
	assert.True(t, results.Functions["TestFunction1"].Result.(bool))
	assert.Equal(t, "Hello world 2!", results.Functions["TestFunction2"].Logs)
	assert.Error(t, results.Functions["FailFunction"].Error)
	assert.True(t, strings.Contains(results.Functions["FailFunction"].Error.Error(), "Unexpected"))
	assert.NotEqual(t, "", results.String())
}
