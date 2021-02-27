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
	assert.Equal(t, "Empty function name", results.Functions[""].Errors[0].Error())
	assert.Equal(t, "Empty function body", results.Functions[""].Errors[1].Error())
	assert.Equal(t, "Empty function body", results.Functions["NoBody"].Errors[0].Error())
	assert.Equal(t, 0, len(results.Functions["GoodFunction"].Errors))
	// fmt.Printf("Results: %s\n", results.String())
	// assert.Fail(t, "Meh")

}
