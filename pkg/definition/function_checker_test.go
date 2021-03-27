package definition_test

import (
	"context"
	_ "embed"
	"github.com/zefhemel/matterless/pkg/definition"
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
	defer sb.Close()
	results := definition.TestDeclarations(defs, sb)
	assert.NoError(t, results.Functions["TestFunction1"])
	assert.NoError(t, results.Functions["TestFunction2"])
	assert.Error(t, results.Functions["FailFunction"])
	assert.Contains(t, results.Functions["FailFunction"].Error(), "Unexpected")
	assert.NotEqual(t, "", results.String())

	ri, err := sb.Function(context.Background(), "TestFunction2", map[string]string{}, defs.ModulesForLanguage("javascript"), defs.Functions["TestFunction2"].Code)
	assert.NoError(t, err)
	result, err := ri.Invoke(context.Background(), struct{}{})
	assert.NoError(t, err)
	assert.Equal(t, "Sup", result)
}
