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

func TestNodeInterpreter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	defs, err := definition.Parse(runtimeCheckMd)
	assert.NoError(t, err)
	sb := sandbox.NewDockerSandbox(0, 0)
	// Flush logs
	go func() {
		for range sb.Logs() {

		}
	}()
	defer sb.Close()
	results := definition.TestDeclarations(defs, sb)
	assert.NoError(t, results.Functions["TestFunction1"])
	assert.NoError(t, results.Functions["TestFunction2"])
	assert.Error(t, results.Functions["FailFunction"])
	assert.True(t, strings.Contains(results.Functions["FailFunction"].Error(), "Unexpected"))
	assert.NotEqual(t, "", results.String())
}
