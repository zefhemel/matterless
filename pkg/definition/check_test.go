package definition_test

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
)

//go:embed test/fail_test.md
var testMd string

func TestCheck(t *testing.T) {
	declarations, err := definition.Parse(testMd)
	assert.NoError(t, err)
	results := definition.Check(declarations)
	assert.Equal(t, "Empty function name", results.Functions[""][0].Error())
	assert.Equal(t, "Empty function body", results.Functions[""][1].Error())
	assert.Equal(t, "Empty function body", results.Functions["NoBody"][0].Error())
	assert.Equal(t, 0, len(results.Functions["GoodFunction"]))

	assert.Equal(t, "no 'token' specified", results.MattermostClients["NoToken"][0].Error())

	assert.Equal(t, "no 'path' defined for endpoint", results.APIGateways["NoHTTPPath"][0].Error())
	assert.Equal(t, "invalid 'schedule' format: Expected 5 to 6 fields, found 1: bla", results.Crons["TestCron"][0].Error())
}
