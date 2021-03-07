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
	decls, err := declaration.Parse(test1Md)
	decls.Normalize()
	assert.NoError(t, err)
	assert.Equal(t, "TestFunction1", decls.Functions["TestFunction1"].Name)
	assert.Equal(t, "TestFunction2", decls.Functions["TestFunction2"].Name)
	assert.Equal(t, "JavaScript", decls.Functions["TestFunction2"].Language)
	assert.Equal(t, "1234", decls.Sources["Me"].Token)
	assert.Equal(t, "posted", decls.Subscriptions["TestSubscription"].EventTypes[0])
	assert.Equal(t, "http://localhost:8065", decls.Environment["MattermostURL"])
	assert.Equal(t, "1234", decls.Environment["MattermostToken"])
}
