package cluster_test

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cluster"
)

func TestMasterElection(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	a := assert.New(t)
	conn, err := cluster.ConnectOrBoot("nats://localhost:4222")
	a.Nil(err)
	defer conn.Close()

	nodes := make([]*cluster.LeaderElection, 50)
	for i := 0; i < len(nodes); i++ {
		nodes[i], err = cluster.NewLeaderElection(conn, "test-me", 500*time.Millisecond)
		a.NoError(err)
	}

	time.Sleep(2 * time.Second)

	leaders := 0
	var leader *cluster.LeaderElection
	for _, node := range nodes {
		if node.IsLeader() {
			leaders++
			leader = node
		}
	}
	a.Equal(1, leaders)
	leader.Close()

	time.Sleep(3 * time.Second)
	leaders = 0
	oldLeader := leader
	for _, node := range nodes {
		if node.IsLeader() {
			leaders++
			leader = node
		}
	}
	a.Equal(1, leaders)
	a.NotEqual(leader.ID, oldLeader.ID)

	// t.Fail()
}
