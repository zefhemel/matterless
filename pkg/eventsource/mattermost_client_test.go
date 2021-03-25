package eventsource_test

import (
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"os"
	"reflect"
	"testing"
)

func TestMatterMostSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	mmSource, err := eventsource.NewMatterMostSource("test", &definition.MattermostClientDef{
		URL:   os.Getenv("mm_test_url"),
		Token: os.Getenv("mm_test_token"),
		Events: map[string][]definition.FunctionID{
			"hello": {"HelloFunction"},
		},
	}, func(name definition.FunctionID, event interface{}) interface{} {
		log.Info("Called", name, event)
		assert.IsType(t, reflect.TypeOf(model.WebSocketEvent{}), reflect.TypeOf(event))
		return struct{}{}
	})
	defer mmSource.Close()
	assert.NoError(t, err)
}
