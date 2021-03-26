package definition_test

import (
	"strings"
	"testing"

	_ "embed"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
)

//go:embed test/test1.md
var test1Md string

func TestValidation(t *testing.T) {
	err := definition.Validate("schema/mattermost_client.schema.json", `
url: http://localhost
token: abc
`)
	assert.NoError(t, err)
	err = definition.Validate("schema/mattermost_client.schema.json", `
url: http://localhost
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

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
	assert.Equal(t, "javascript", decls.Modules["my-module"].Language)
	assert.Equal(t, "/test", decls.APIs[0].Path)
	assert.Equal(t, definition.FunctionID("TestFunction2"), decls.APIs[0].Function)
	assert.Equal(t, definition.FunctionID("TestFunction1"), decls.Bots["MyBot"].Events["posted"][0])

	assert.Equal(t, "0 * * * * *", decls.Crons[0].Schedule)
	assert.Equal(t, definition.FunctionID("MyRepeatedTask"), decls.Crons[0].Function)
}

func TestParseFailures(t *testing.T) {
	_, err := definition.Parse(strings.ReplaceAll(`# MattermostClient: MyClient
|||
url: http://bla.com
|||
`, "|||", "```"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token")
	_, err = definition.Parse(strings.ReplaceAll(`# API
|||
- path: /bla
|||
`, "|||", "```"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "function is required")
}
