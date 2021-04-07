package eventbus_test

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"testing"
	"time"
)

func TestLocalEventBus(t *testing.T) {
	a := assert.New(t)
	eb := eventbus.NewLocalEventBus()
	eb.Publish("testEventNoListeners", struct{}{})
	catchAllCallback := func(eventName string, eventData interface{}) {
		log.Infof("Got catch-all event: '%s' with data: %+v", eventName, eventData)
	}
	eb.Subscribe("*", catchAllCallback)
	receivedRandom := false
	randomCallback := func(eventName string, eventData interface{}) {
		receivedRandom = true
	}
	eb.Subscribe("random", randomCallback)
	eb.Publish("random", struct{}{})
	a.True(receivedRandom, "received random")
	eb.Unsubscribe("random", randomCallback)
	receivedRandom = false
	eb.Publish("random", struct{}{})
	a.False(receivedRandom, "unsubscribed random")

	called := false
	called2 := false
	eb.Subscribe("myApp:something", func(eventName string, eventData interface{}) {
		called = true
	})
	eb.Subscribe("myApp:something", func(eventName string, eventData interface{}) {
		called2 = true
	})
	eb.Publish("myApp:something", struct{}{})
	a.True(called)
	a.True(called2)
	called = false
	called2 = false

	eb.Subscribe("http:GET:/hello", func(eventName string, eventData interface{}) {
		eb.Respond(eventData, eventData)
	})

	myEvent := map[string]interface{}{
		"name": "pete",
	}
	r, err := eb.Request("http:GET:/hello", myEvent, time.Second)
	a.NoError(err)
	a.Equal(myEvent, r)

	eb.Unsubscribe("*", catchAllCallback)

	r, err = eb.Request("http:GET:/non-existing", myEvent, time.Second)
	a.Equal(eventbus.NoListenersError, err)
}
