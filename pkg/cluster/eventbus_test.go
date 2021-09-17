package cluster_test

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/config"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cluster"
)

func TestNatsCluster(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	a := assert.New(t)
	eb, err := cluster.ConnectOrBoot(&config.Config{
		DataDir:        "nats-data",
		ClusterNatsUrl: "nats://localhost:4222",
	})
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

	ceb.SubscribeFetchClusterInfo(func() *cluster.NodeInfo {
		return &cluster.NodeInfo{
			ID:   1,
			Apps: map[string]*cluster.AppInfo{},
		}
	})
	ceb.SubscribeFetchClusterInfo(func() *cluster.NodeInfo {
		return &cluster.NodeInfo{
			ID:   2,
			Apps: map[string]*cluster.AppInfo{},
		}
	})

	inf, err := ceb.FetchClusterInfo(1 * time.Second)
	a.NoError(err)
	a.Equal(cluster.NodeID(1), inf.Nodes[1].ID)
	a.Equal(cluster.NodeID(2), inf.Nodes[2].ID)

	// t.Fail()
}
