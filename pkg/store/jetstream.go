package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/util"

	log "github.com/sirupsen/logrus"
)

const sequenceKey = "$$seq"

var SyncTimeoutErr = errors.New("sync timeout")

type syncMessage struct {
	ID string `json:"id"`
}

type PutMessage struct {
	ID    string      `json:"id"`
	Key   string      `json:"k"`
	Value interface{} `json:"v"`
}

type DeleteMessage struct {
	ID  string `json:"id"`
	Key string `json:"k"`
}

// JetstreamStore implements the Store interface on top of a NATS connection using NATS JetStream
// It does so by publishing all writes (put and delete) to a stream persisted stream
// Every peer connects to this stream. Upon first connect, all events are replayed, if the peer
// has connected before, it will resume the stream from whenever it had left off
// All changes are then replicated locally in the wrappedStore (usually you'd use a LevelDB for this)
// Lookups all happen instantly locally
// Writes are first published to the stream, and only persisted locally after the are received back from the stream
// Write calls only return when the event has "come back"
// Reads are eventually consistent, to force a synced state use the `.Sync()` call
type JetstreamStore struct {
	conn            *nats.Conn
	js              nats.JetStreamContext
	streamName      string
	localCacheStore Store

	// Event names
	syncEvent   string
	putEvent    string
	deleteEvent string

	ackMutex   sync.Mutex
	ackWaiting map[string]chan struct{}

	// Sync
	syncMessageSeq  string
	syncMessageChan chan struct{}

	listenCancel context.CancelFunc
}

var _ Store = &JetstreamStore{}

func NewJetstreamStore(conn *nats.Conn, streamName string, wrappedStore Store) (*JetstreamStore, error) {
	var err error
	jss := &JetstreamStore{
		conn:            conn,
		streamName:      streamName,
		localCacheStore: wrappedStore,
		syncEvent:       fmt.Sprintf("%s.sync", streamName),
		putEvent:        fmt.Sprintf("%s.put", streamName),
		deleteEvent:     fmt.Sprintf("%s.delete", streamName),
		ackWaiting:      make(map[string]chan struct{}),
	}
	jss.js, err = conn.JetStream()
	if err != nil {
		return nil, err
	}

	if err := jss.init(); err != nil {
		return nil, err
	}

	return jss, nil
}

func (jss *JetstreamStore) init() error {
	_, err := jss.js.StreamInfo(jss.streamName)
	if err != nil {
		// Likely does not exist yet, let's create it
		_, err = jss.js.AddStream(&nats.StreamConfig{
			Name:     jss.streamName,
			Subjects: []string{fmt.Sprintf("%s.*", jss.streamName)},
			Storage:  nats.FileStorage,
		})

		if err != nil {
			return errors.Wrap(err, "stream create")
		}
	}
	return nil
}

func (jss *JetstreamStore) Connect(timeout time.Duration) error {
	// By default we replay all events
	syncOpt := nats.DeliverAll()

	// ... except if we don't have to
	startSeq, _ := jss.localCacheStore.Get(sequenceKey)
	if startFlt, ok := startSeq.(float64); ok {
		// No need to deliver all, let's start later
		syncOpt = nats.StartSequence(uint64(startFlt))
		// log.Infof("Starting at %d", uint64(startFlt))
	}

	// Subscribe to all events based on syncOpt
	sub, err := jss.js.SubscribeSync(fmt.Sprintf("%s.*", jss.streamName), nats.AckNone(), syncOpt)
	if err != nil {
		return errors.Wrap(err, "subscription")
	}

	// Context for stopping our go-routine later
	var ctx context.Context
	ctx, jss.listenCancel = context.WithCancel(context.Background())

	// Already, all set, let's start listenin'
	go jss.receiveMessages(ctx, sub)

	if err := jss.Sync(timeout); err != nil {
		return errors.Wrap(err, "sync")
	}

	return nil
}

// Sends out a sync message and waits for it to come back to ensure a fully synced state
func (jss *JetstreamStore) Sync(timeout time.Duration) error {
	// To figure out when we've processed the backlog of messages we're going to publish a dummy "sync" message
	// and wait for it to come back to us
	sm := syncMessage{
		ID: uuid.NewString(),
	}
	jss.syncMessageChan = make(chan struct{})
	jss.syncMessageSeq = sm.ID

	_, err := jss.js.Publish(jss.syncEvent, util.MustJsonByteSlice(sm))
	if err != nil {
		return errors.Wrap(err, "publish sync")
	}

	// And return when we got our "sync" message back
	select {
	case <-time.After(timeout):
		return SyncTimeoutErr
	case <-jss.syncMessageChan:
	}

	return nil
}

func (jss *JetstreamStore) receiveMessages(ctx context.Context, sub *nats.Subscription) {
loop:
	for {
		m, err := sub.NextMsgWithContext(ctx)

		if err == context.Canceled {
			// This is fine, .Close() called
			// log.Info("Shutting down")
			return
		}
		if err != nil {
			log.Errorf("Error receiving message: %s", err)
			return
		}

		meta, _ := m.Metadata()

		switch m.Subject {
		case jss.syncEvent:
			if jss.syncMessageSeq == "" {
				// Not waiting for a sync, skip
				continue loop
			}
			var syncMessage syncMessage
			if err := json.Unmarshal(m.Data, &syncMessage); err != nil {
				log.Errorf("Could not unmarshal sync message: %s", err)
				continue loop
			}
			// log.Infof("Received a sync event: %s, waiting for %s", syncMessage.ID, jss.syncMessageSeq)
			if syncMessage.ID == jss.syncMessageSeq {
				// log.Info("Gotcha!")
				jss.syncMessageSeq = ""
				close(jss.syncMessageChan)
			}
		case jss.putEvent:
			// log.Infof("Received a put message: %s\n", string(m.Data))
			var putMessage PutMessage
			if err := json.Unmarshal(m.Data, &putMessage); err != nil {
				log.Errorf("Could not unmarshal put message: %s", err)
				continue loop
			}

			// Persist locally
			if err := jss.localCacheStore.Put(putMessage.Key, putMessage.Value); err != nil {
				log.Errorf("Could not persist put: %s", err)
				continue loop
			}

			// Persist sequence number to local store
			if err := jss.localCacheStore.Put(sequenceKey, meta.Sequence.Stream); err != nil {
				log.Errorf("Could not persist sequence number: %s", err)
				continue loop
			}

			// Acknowledge receiving in case this was sent locally
			// log.Info("Put ID ", putMessage.ID)
			jss.ackMutex.Lock()
			if ackCh, ok := jss.ackWaiting[putMessage.ID]; ok {
				ackCh <- struct{}{}
				delete(jss.ackWaiting, putMessage.ID)
			}
			jss.ackMutex.Unlock()

		case jss.deleteEvent:
			// log.Infof("Received a put message: %s\n", string(m.Data))
			var deleteMessage DeleteMessage
			if err := json.Unmarshal(m.Data, &deleteMessage); err != nil {
				log.Errorf("Could not unmarshal delete message: %s", err)
				continue loop
			}

			// Persist locally
			if err := jss.localCacheStore.Delete(deleteMessage.Key); err != nil {
				log.Errorf("Could not persist delete: %s", err)
				continue loop
			}

			// Persist sequence number to local store
			if err := jss.localCacheStore.Put(sequenceKey, meta.Sequence.Stream); err != nil {
				log.Errorf("Could not persist sequence number: %s", err)
				continue loop
			}

			// Acknowledge receiving in case this was sent locally
			// log.Info("Delete ID ", deleteMessage.ID)
			jss.ackMutex.Lock()
			if ackCh, ok := jss.ackWaiting[deleteMessage.ID]; ok {
				ackCh <- struct{}{}
				delete(jss.ackWaiting, deleteMessage.ID)
			}
			jss.ackMutex.Unlock()
		}
	}
}

func (jss *JetstreamStore) Disconnect() {
	jss.syncMessageSeq = ""
	jss.syncMessageChan = nil
	jss.listenCancel()
}

func (jss *JetstreamStore) DeleteStore() error {
	if err := jss.localCacheStore.DeleteStore(); err != nil {
		return err
	}
	if err := jss.js.DeleteStream(jss.streamName); err != nil {
		if err == nats.ErrStreamNotFound {
			// Probably somebody else came first, that's ok
			return nil
		} else {
			return err
		}
	}
	return nil
}

func (jss *JetstreamStore) Put(key string, val interface{}) error {
	pm := PutMessage{
		ID:    uuid.NewString(),
		Key:   key,
		Value: val,
	}
	ch := make(chan struct{})
	jss.ackMutex.Lock()
	jss.ackWaiting[pm.ID] = ch
	jss.ackMutex.Unlock()
	_, err := jss.js.Publish(jss.putEvent, util.MustJsonByteSlice(pm))
	if err != nil {
		return nil
	}
	<-ch
	return nil
}

func (jss *JetstreamStore) Delete(key string) error {
	dm := DeleteMessage{
		ID:  uuid.NewString(),
		Key: key,
	}
	ch := make(chan struct{})
	jss.ackMutex.Lock()
	jss.ackWaiting[dm.ID] = ch
	jss.ackMutex.Unlock()
	_, err := jss.js.Publish(jss.deleteEvent, util.MustJsonByteSlice(dm))
	if err != nil {
		return nil
	}
	<-ch
	return nil
}

func (jss *JetstreamStore) Get(key string) (interface{}, error) {
	return jss.localCacheStore.Get(key)
}

func (jss *JetstreamStore) QueryRange(startKey string, endKey string) ([]QueryResult, error) {
	return jss.localCacheStore.QueryRange(startKey, endKey)
}

func (jss *JetstreamStore) QueryPrefix(prefix string) ([]QueryResult, error) {
	return jss.localCacheStore.QueryPrefix(prefix)
}

func (jss *JetstreamStore) Close() error {
	return jss.localCacheStore.Close()
}

// Subscribe to any new PUT events
func (jss *JetstreamStore) SubscribePuts(callback func(event PutMessage)) (*nats.Subscription, error) {
	return jss.js.Subscribe(jss.putEvent, func(msg *nats.Msg) {
		var pm PutMessage
		if err := json.Unmarshal(msg.Data, &pm); err != nil {
			log.Errorf("Could not unmarshal data: %s", err)
			return
		}
		callback(pm)
	}, nats.DeliverNew())
}

// Subscribe to any new DELETE events
func (jss *JetstreamStore) SubscribeDeletes(callback func(event DeleteMessage)) (*nats.Subscription, error) {
	return jss.js.Subscribe(jss.deleteEvent, func(msg *nats.Msg) {
		var dm DeleteMessage
		if err := json.Unmarshal(msg.Data, &dm); err != nil {
			log.Errorf("Could not unmarshal data: %s", err)
			return
		}
		callback(dm)
	}, nats.DeliverNew())
}
