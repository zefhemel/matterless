package store_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/store"
	"testing"
	"time"
)

func TestEventedStore(t *testing.T) {
	eb := eventbus.NewLocalEventBus()
	s := &store.MockStore{}
	es := store.NewEventedStore(s, eb)

	putTestChan := make(chan interface{})
	eb.Subscribe("store:put:TestKey", func(eventName string, eventData interface{}) {
		putTestChan <- eventData.(*store.PutEvent).NewValue
	})
	assert.NoError(t, es.Put("TestKey", "Sup"))
	select {
	case putVal := <-putTestChan:
		assert.Equal(t, "Sup", putVal)
	case <-time.After(1 * time.Second):
		assert.Fail(t, "Event timeout")
	}

	delChan := make(chan struct{})
	eb.Subscribe("store:del:TestKey", func(eventName string, eventData interface{}) {
		close(delChan)
	})
	assert.NoError(t, es.Delete("TestKey"))
	select {
	case <-delChan:
	case <-time.After(1 * time.Second):
		assert.Fail(t, "Event timeout")
	}
}
