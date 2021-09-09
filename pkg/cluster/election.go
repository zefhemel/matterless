package cluster

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
)

const (
// broadcastInterval = 2 * time.Second
// timeoutDuration   = 5 * time.Second
)

type NodeID = uint64

type LeaderElection struct {
	conn                  *nats.Conn
	heartbeatEvent        string
	heartbeatSubscription *nats.Subscription

	startTime         time.Time
	broadcastInterval time.Duration
	timeoutDuration   time.Duration

	ID   NodeID
	done bool

	mutex     sync.Mutex
	hearbeats map[NodeID]time.Time
}

type heartbeatMessage struct {
	NodeID NodeID
}

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func NewLeaderElection(conn *nats.Conn, hearthbeatEvent string, broadcastInterval time.Duration) (*LeaderElection, error) {
	var err error
	le := &LeaderElection{
		heartbeatEvent:    hearthbeatEvent,
		hearbeats:         make(map[NodeID]time.Time),
		startTime:         time.Now(),
		broadcastInterval: broadcastInterval,
		timeoutDuration:   broadcastInterval * 3,
		conn:              conn,
	}

	le.heartbeatSubscription, err = conn.Subscribe(hearthbeatEvent, func(msg *nats.Msg) {
		var hm heartbeatMessage
		if err := json.Unmarshal(msg.Data, &hm); err != nil {
			log.Errorf("Could not unmarshal heartbeat message")
		}
		// log.Infof("Got heartbeat %d", hm.NodeID)
		le.mutex.Lock()
		defer le.mutex.Unlock()
		le.hearbeats[hm.NodeID] = time.Now()
		if le.ID == 0 {
			// Time for me to generate my own ID
			le.ID = generateNodeID(hm.NodeID)
			go le.broadcaster()
		}
	})

	// Async kick off initial election
	go le.Leader()

	if err != nil {
		return nil, errors.Wrap(err, "heartbeat subscription")
	}

	return le, nil
}

func (le *LeaderElection) Leader() NodeID {
	le.mutex.Lock()
	defer le.mutex.Unlock()
	if len(le.hearbeats) == 0 {
		// log.Info("No heartbets received yet, let's wait")
		// No heartbeats received, two options:
		// 1. Too early, need to wait or
		// 2. I'm the first one in the cluster
		le.mutex.Unlock()
		// Wait up to timeOutDuration
		for time.Since(le.startTime) < le.timeoutDuration && le.ID == 0 {
			time.Sleep(time.Duration(rand.Int63n(500)) * time.Millisecond)
		}
		le.mutex.Lock()
		if len(le.hearbeats) == 0 {
			// log.Info("Still nuthin, I suppose I'm first?")
			// Ok I'm the first, cool
			le.ID = generateNodeID(0)
			go le.broadcaster()
			return le.ID
		}
	}
	// Iterate over all heartbeats
	now := time.Now()
	currentLeaderID := NodeID(0)
	for id, t := range le.hearbeats {
		if now.Sub(t) > le.timeoutDuration {
			log.Infof("Ejecting %d", id)
			delete(le.hearbeats, id)
			continue
		}
		if currentLeaderID == 0 || currentLeaderID > id {
			currentLeaderID = id
		}
	}
	return currentLeaderID
}

func (le *LeaderElection) IsLeader() bool {
	return le.Leader() == le.ID
}

func generateNodeID(min NodeID) NodeID {
	id := r.Uint64()
	for id <= min {
		id = r.Uint64()
	}
	return id
}

func (le *LeaderElection) broadcaster() {
	for {
		if le.done {
			log.Infof("Disconnected %d", le.ID)
			return
		}
		// log.Infof("Broadcasting my ID %d", le.ID)
		if err := le.conn.Publish(le.heartbeatEvent, util.MustJsonByteSlice(heartbeatMessage{le.ID})); err != nil {
			log.Errorf("Could not broadcast heartbeat: %s", err)
		}
		time.Sleep(le.broadcastInterval)
	}
}

func (le *LeaderElection) Close() error {
	le.done = true
	return le.heartbeatSubscription.Unsubscribe()
}
