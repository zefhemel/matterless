package definition_test

import (
	_ "embed"
	"github.com/zefhemel/matterless/pkg/definition"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

//go:embed test/runtime_check.md
var runtimeCheckMd string

func TestInterpreter(t *testing.T) {
	defs, err := definition.Parse(runtimeCheckMd)
	assert.NoError(t, err)
	sb := sandbox.NewMockSandbox(nil, "Ok", nil)
	results := definition.TestDeclarations(defs, sb)
	assert.Equal(t, nil, results.Functions["TestFunction1"].Error)
}

func TestNodeInterpreter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	defs, err := definition.Parse(runtimeCheckMd)
	assert.NoError(t, err)
	sb := sandbox.NewDockerSandbox(0, 0)
	results := definition.TestDeclarations(defs, sb)
	assert.NoError(t, results.Functions["TestFunction1"].Error)
	assert.Equal(t, "Hello world!", results.Functions["TestFunction1"].Logs)
	assert.True(t, results.Functions["TestFunction1"].Result.(bool))
	assert.Equal(t, "Hello world 2!", results.Functions["TestFunction2"].Logs)
	assert.Error(t, results.Functions["FailFunction"].Error)
	assert.True(t, strings.Contains(results.Functions["FailFunction"].Error.Error(), "Unexpected"))
	assert.NotEqual(t, "", results.String())
}
