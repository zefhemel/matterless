package declaration_test

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/declaration"
)

//go:embed test/fail_test.md
var testMd string

func TestCheck(t *testing.T) {
	declarations, err := declaration.Parse([]string{testMd})
	assert.NoError(t, err)
	results := declaration.Check(declarations)
	assert.Equal(t, "Empty function name", results.Functions[""][0].Error())
	assert.Equal(t, "Empty function body", results.Functions[""][1].Error())
	assert.Equal(t, "Empty function body", results.Functions["NoBody"][0].Error())
	assert.Equal(t, 0, len(results.Functions["GoodFunction"]))

	assert.Equal(t, "no Token specified", results.Sources["NoToken"][0].Error())
	assert.Equal(t, "no Function to trigger specified", results.Subscriptions["NoFunction"][0].Error())
	assert.Equal(t, "function TestFunction2 not found", results.Subscriptions["NonExistingFunction"][0].Error())
	assert.Equal(t, "subscription source NoSource not found", results.Subscriptions["NonExistingSource"][0].Error())

}
