package cluster_test

import (
	"sync"
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
	var wg sync.WaitGroup
	for i := 0; i < len(nodes); i++ {
		j := i
		wg.Add(1)
		go func() {
			nodes[j], err = cluster.NewLeaderElection(conn, "election.heartbeat", "election.getLeader", 500*time.Millisecond)
			a.NoError(err)
			wg.Done()
		}()
	}
	wg.Wait()

	time.Sleep(600 * time.Millisecond)

	leaders := 0
	var leader *cluster.LeaderElection
	for _, node := range nodes {
		if node.IsLeader() {
			leaders++
			log.Infof("Found leader: %d", node.ID)
			leader = node
		}
	}
	a.Equal(1, leaders)

	log.Info("Now kicking out current leader")
	leader.Close()

	time.Sleep(3 * time.Second)
	leaders = 0
	oldLeader := leader
	for _, node := range nodes {
		if node.IsLeader() {
			leaders++
			log.Infof("Found leader: %d", node.ID)
			leader = node
		}
	}
	a.Equal(1, leaders)
	a.NotEqual(leader.ID, oldLeader.ID)

	// t.Fail()
}
