package definition_test

import (
	"testing"

	_ "embed"

	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
)

//go:embed test/test1.md
var test1Md string

func TestParser(t *testing.T) {
	posts := []*model.Post{
		{
			Message: test1Md,
		},
	}
	def, err := definition.Parse(posts)
	assert.NoError(t, err)
	assert.Equal(t, "TestFunction1", def.Functions["TestFunction1"].Name)
	assert.Equal(t, "TestFunction2", def.Functions["TestFunction2"].Name)
	assert.Equal(t, "JavaScript", def.Functions["TestFunction2"].Language)
	assert.Equal(t, "1234", def.Sources["Me"].Token)
	assert.Equal(t, "Off-Topic", def.Subscriptions["TestSubscription"].Channel)
	assert.Equal(t, "posted", def.Subscriptions["TestSubscription"].EventTypes[0])
}
