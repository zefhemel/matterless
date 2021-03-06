package declaration_test

import (
	"testing"

	_ "embed"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/declaration"
)

//go:embed test/test1.md
var test1Md string

func TestParser(t *testing.T) {
	def, err := declaration.Parse([]string{test1Md})
	assert.NoError(t, err)
	assert.Equal(t, "TestFunction1", def.Functions["TestFunction1"].Name)
	assert.Equal(t, "TestFunction2", def.Functions["TestFunction2"].Name)
	assert.Equal(t, "JavaScript", def.Functions["TestFunction2"].Language)
	assert.Equal(t, "1234", def.Sources["Me"].Token)
	assert.Equal(t, "posted", def.Subscriptions["TestSubscription"].EventTypes[0])
}
