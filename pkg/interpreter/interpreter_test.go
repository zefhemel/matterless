package interpreter_test

import (
	_ "embed"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/interpreter"
	"github.com/zefhemel/matterless/pkg/sandbox"
)

//go:embed test/test1.md
var test1Md string

func TestInterpreter(t *testing.T) {
	posts := []*model.Post{
		{
			Message: test1Md,
		},
	}
	defs, err := definition.Parse(posts)
	assert.NoError(t, err)
	sb := sandbox.NewMockSandbox(nil, []string{"Ok"}, nil)
	results := interpreter.TestDefinitions(defs, sb)
	assert.Equal(t, nil, results.FunctionResults["TestFunction1"].Error)
}
