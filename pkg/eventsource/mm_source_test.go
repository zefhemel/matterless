package eventsource_test

import (
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"os"
	"reflect"
	"testing"
)

func DisabledTestNewMatterMostSource(t *testing.T) {
	mmSource, err := eventsource.NewMatterMostSource(os.Getenv("mm_test_url"), os.Getenv("mm_test_token"))
	assert.NoError(t, err)
	assert.NoError(t, mmSource.Start())
	evt := <-mmSource.Events()
	assert.IsType(t, reflect.TypeOf(model.WebSocketEvent{}), reflect.TypeOf(evt))
	//log.Info(evt)
	mmSource.Stop()
	_, ok := <-mmSource.Events()
	assert.Equal(t, false, ok, "event stream should be closed at this point")
	// Start again
	assert.NoError(t, mmSource.Start())
	evt = <-mmSource.Events()
	assert.IsType(t, reflect.TypeOf(model.WebSocketEvent{}), reflect.TypeOf(evt))
	log.Info(evt)
	mmSource.Stop()
}
