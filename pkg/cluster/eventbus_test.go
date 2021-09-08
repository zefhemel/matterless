package cluster_test

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cluster"
)

func TestNatsCluster(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	a := assert.New(t)
	eb, err := cluster.ConnectOrBoot("nats://localhost:4225")
	a.Nil(err)
	defer eb.Close()

	ceb := cluster.NewClusterEventBus(eb, "test")
	ceb.SubscribeInvokeFunction("callme", func(event interface{}) (interface{}, error) {
		fmt.Println("Got this on the succeeding one", event)
		return "OK", nil
	})
	ceb.SubscribeInvokeFunction("callme", func(event interface{}) (interface{}, error) {
		fmt.Println("Got this on the failing one", event)
		return nil, errors.New("FAIL")
	})

	for i := 0; i < 10; i++ {
		res, err := ceb.InvokeFunction("callme", map[string]interface{}{
			"name": "Pete",
		})
		if err == nil && res != "OK" {
			a.Fail("No error incorrect response")
		} else if err != nil && err.Error() != "FAIL" {
			a.Fail("Error, incorrect message")
		}
	}

	// t.Fail()
}
