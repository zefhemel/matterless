package definition_test

import (
	"testing"

	_ "embed"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
)

//go:embed test/test1.md
var test1Md string

func TestParser(t *testing.T) {
	decls, err := definition.Parse(test1Md)
	decls.Normalize()
	assert.NoError(t, err)
	assert.Equal(t, "TestFunction1", decls.Functions["TestFunction1"].Name)
	assert.Equal(t, "TestFunction2", decls.Functions["TestFunction2"].Name)
	assert.Equal(t, "javascript", decls.Functions["TestFunction2"].Language)
	assert.Equal(t, "1234", decls.MattermostClients["Me"].Token)
	assert.Equal(t, "http://localhost:8065", decls.Environment["MattermostURL"])
	assert.Equal(t, "1234", decls.Environment["MattermostToken"])
	assert.Equal(t, "javascript", decls.Libraries[""].Language)
	assert.Equal(t, "/test", decls.APIGateways["MyHTTP"].Endpoints[0].Path)
	assert.Equal(t, definition.FunctionID("TestFunction2"), decls.APIGateways["MyHTTP"].Endpoints[0].Function)
	assert.Equal(t, definition.FunctionID("TestFunction1"), decls.Bots["MyBot"].Events["posted"][0])

	assert.Equal(t, "0 * * * * *", decls.Crons["EveryMinute"].Schedule)
	assert.Equal(t, definition.FunctionID("MyRepeatedTask"), decls.Crons["EveryMinute"].Function)
}
