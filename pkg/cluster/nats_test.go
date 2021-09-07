package cluster_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/util"
)

func TestNatsCluster(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	a := assert.New(t)
	eb, err := cluster.ConnectOrBoot("nats://localhost:4225")
	a.Nil(err)
	defer eb.Close()

	ceb := cluster.NewClusterEventBus(eb, "test")
	ceb.QueueSubscribe("callme", "callme.q", func(msg *nats.Msg) {
		fmt.Println("Got this 1", string(msg.Data))
		a.Nil(msg.Respond([]byte("OK")))
	})
	ceb.QueueSubscribe("callme", "callme.q", func(msg *nats.Msg) {
		fmt.Println("Got this 2", string(msg.Data))
		a.Nil(msg.Respond([]byte("OK")))
	})

	for i := 0; i < 10; i++ {
		res, err := ceb.Request("callme", util.MustJsonByteSlice(map[string]interface{}{
			"name": "Pete",
		}), 1*time.Second)
		a.Nil(err)
		a.Equal([]byte("OK"), res.Data)
	}

	// time.Sleep(1 * time.Second)
	// t.Fail()
}
