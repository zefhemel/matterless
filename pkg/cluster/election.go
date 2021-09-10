package cluster

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
)

type NodeID = uint64

type LeaderElection struct {
	conn                         *nats.Conn
	currentLeaderRPCSubject      string
	heartbeatSubject             string
	heartbeatSubscription        *nats.Subscription
	currentLeaderRPCSubscription *nats.Subscription

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

func NewLeaderElection(conn *nats.Conn, currentLeaderRPCSubject string, hearthbeatSubject string, broadcastInterval time.Duration) (*LeaderElection, error) {
	var err error
	le := &LeaderElection{
		currentLeaderRPCSubject: currentLeaderRPCSubject,
		heartbeatSubject:        hearthbeatSubject,
		hearbeats:               make(map[NodeID]time.Time),
		startTime:               time.Now(),
		broadcastInterval:       broadcastInterval,
		timeoutDuration:         broadcastInterval * 3,
		conn:                    conn,
	}

	// Let's make a quick getLeader call to the bus and see if somebody got a response
	resp, err := conn.Request(currentLeaderRPCSubject, []byte{}, 2*time.Second)
	if err == nil {
		// Got answer, great
		var msg heartbeatMessage
		if err := json.Unmarshal(resp.Data, &msg); err != nil {
			return nil, errors.Wrap(err, "unmarshall getLeader")
		}
		// Let's persist it
		le.hearbeats[msg.NodeID] = time.Now()
		// And generate an ID > the current leader's ID
		le.ID = generateNodeID(msg.NodeID)
	} else if err == nats.ErrNoResponders || err == nats.ErrTimeout {
		// Either no responders or timeout => We're the first!
		// log.Infof("Got RPC error: %s", err)
		// Generate a random ID
		le.ID = generateNodeID(0)
		// Let's persist it
		le.hearbeats[le.ID] = time.Now()
	} else {
		return nil, errors.Wrap(err, "getLeader call")
	}
	// And let's start broadcasting
	go le.broadcaster()

	le.currentLeaderRPCSubscription, err = conn.QueueSubscribe(currentLeaderRPCSubject, fmt.Sprintf("%s.workers", currentLeaderRPCSubject), func(msg *nats.Msg) {
		msg.Respond(util.MustJsonByteSlice(heartbeatMessage{le.Leader()}))
	})
	if err != nil {
		return nil, errors.Wrap(err, "getLeader subscription")
	}

	le.heartbeatSubscription, err = conn.Subscribe(hearthbeatSubject, func(msg *nats.Msg) {
		var hm heartbeatMessage
		if err := json.Unmarshal(msg.Data, &hm); err != nil {
			log.Errorf("Could not unmarshal heartbeat message")
		}
		// log.Infof("Got heartbeat %d", hm.NodeID)
		le.mutex.Lock()
		defer le.mutex.Unlock()
		le.hearbeats[hm.NodeID] = time.Now()
	})

	if err != nil {
		return nil, errors.Wrap(err, "heartbeat subscription")
	}

	return le, nil
}

func (le *LeaderElection) Leader() NodeID {
	le.mutex.Lock()
	defer le.mutex.Unlock()
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
	if le.done {
		// Short circuit
		return false
	}
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
		if err := le.conn.Publish(le.heartbeatSubject, util.MustJsonByteSlice(heartbeatMessage{le.ID})); err != nil {
			log.Errorf("Could not broadcast heartbeat: %s", err)
		}
		time.Sleep(le.broadcastInterval)
	}
}

func (le *LeaderElection) Close() error {
	le.done = true
	return le.heartbeatSubscription.Unsubscribe()
}
