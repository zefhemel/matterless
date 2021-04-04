package queue

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/store"
	"sync"
	"time"
)

// Store keys:
// queue:$name:$messageID = body
// queue:$name:_queue []MessageID
// quque:$name:_inflight map[MesasgeID]expiretime

type LocalQueue struct {
	name            string
	eventBus        eventbus.EventBus
	store           store.Store
	queueLock       sync.Mutex
	inFlightLock    sync.Mutex
	inflightTimeout time.Duration
	done            chan struct{}
}

var _ Queue = &LocalQueue{}

func NewLocalQueue(store store.Store, name string, inflightTimeout time.Duration) *LocalQueue {
	q := &LocalQueue{
		store:           store,
		name:            name,
		inflightTimeout: inflightTimeout,
		eventBus:        eventbus.NewLocalEventBus(),
		done:            make(chan struct{}),
	}

	go q.cleaner()

	return q
}

func (q *LocalQueue) Send(message *Message) error {
	// Write message to store
	if err := q.putMessage(message); err != nil {
		return errors.Wrap(err, "put message")
	}
	if err := q.queueMessage(message); err != nil {
		return errors.Wrap(err, "queue message")
	}
	return nil
}

func (q *LocalQueue) queueMessage(message *Message) error {
	q.queueLock.Lock()
	defer q.queueLock.Unlock()
	queueKey := fmt.Sprintf("queue:%s:_queue", q.name)
	ids, err := q.store.Get(queueKey)
	if err != nil {
		return errors.Wrap(err, "queue get")
	}
	idStrings := []string{}
	idsSlice, ok := ids.([]interface{})
	if ok {
		for _, idInt := range idsSlice {
			idStrings = append(idStrings, idInt.(string))
		}
	}
	idStrings = append(idStrings, string(message.ID))
	if err := q.store.Put(queueKey, idStrings); err != nil {
		return errors.Wrap(err, "put queue")
	}

	// Publish event
	q.eventBus.PublishAsync("message", struct{}{})
	return nil
}

func (q *LocalQueue) Receive() (*Message, error) {
	q.queueLock.Lock()
	queueKey := fmt.Sprintf("queue:%s:_queue", q.name)
	ids, err := q.store.Get(queueKey)
	if err != nil {
		q.queueLock.Unlock()
		return nil, errors.Wrap(err, "queue get")
	}
	idStrings := []string{}
	idsSlice, ok := ids.([]interface{})
	if ok {
		for _, idInt := range idsSlice {
			idStrings = append(idStrings, idInt.(string))
		}
	}
	if len(idStrings) == 0 {
		q.queueLock.Unlock()
		return nil, NoMessageError
	}
	topID := idStrings[0]
	idStrings = idStrings[1:]
	if err := q.store.Put(queueKey, idStrings); err != nil {
		q.queueLock.Unlock()
		return nil, errors.Wrap(err, "put queue")
	}
	q.queueLock.Unlock()

	queueMessage, err := q.getMessage(MessageID(topID))
	if err != nil {
		return nil, errors.Wrap(err, "lookup message")
	}

	// Put into in-flight
	inflightKey := fmt.Sprintf("queue:%s:_inflight", q.name)
	q.inFlightLock.Lock()
	defer q.inFlightLock.Unlock()
	inflightMessages, err := q.store.Get(inflightKey)
	if err != nil {
		return nil, errors.Wrap(err, "get inflight")
	}
	inflightMap, ok := inflightMessages.(map[string]interface{})
	if !ok {
		// Initializing fresh inflight
		inflightMap = map[string]interface{}{}
	}
	inflightMap[string(queueMessage.ID)] = time.Now().Add(q.inflightTimeout).Format(time.RFC3339)
	if err := q.store.Put(inflightKey, inflightMap); err != nil {
		return nil, errors.Wrap(err, "put inflight")
	}
	return queueMessage, nil
}

func (q *LocalQueue) getMessage(messageID MessageID) (*Message, error) {
	// Lookup message
	messageKey := fmt.Sprintf("queue:%s:%s", q.name, messageID)
	msgIn, err := q.store.Get(messageKey)
	if err != nil {
		return nil, errors.Wrap(err, "message lookup")
	}

	queueMessage := &Message{}
	err = mapstructure.Decode(msgIn, queueMessage)
	if err != nil {
		return nil, errors.Wrap(err, "map structure")
	}
	return queueMessage, nil
}

func (q *LocalQueue) Ack(id MessageID) error {
	inflightKey := fmt.Sprintf("queue:%s:_inflight", q.name)
	q.inFlightLock.Lock()
	defer q.inFlightLock.Unlock()
	// Fetch all inflight messages
	inflightMessages, err := q.store.Get(inflightKey)
	if err != nil {
		return errors.Wrap(err, "get inflight")
	}
	inflightMap, ok := inflightMessages.(map[string]interface{})
	if !ok {
		inflightMap = map[string]interface{}{}
	}
	// Remove inflight message
	delete(inflightMap, string(id))

	// Put back
	if err := q.store.Put(inflightKey, inflightMap); err != nil {
		return errors.Wrap(err, "put inflight")
	}

	// Delete message from store
	if err := q.store.Delete(fmt.Sprintf("queue:%s:%s", q.name, id)); err != nil {
		return errors.Wrap(err, "delete message")
	}
	return nil
}

func (q *LocalQueue) EventBus() eventbus.EventBus {
	return q.eventBus
}

func (q *LocalQueue) Stats() (Stats, error) {
	var queueStats Stats

	// Inflight stats
	inflightMessages, err := q.store.Get(fmt.Sprintf("queue:%s:_inflight", q.name))
	if err != nil {
		return queueStats, errors.Wrap(err, "inflight get")
	}
	inflightMap, ok := inflightMessages.(map[string]interface{})
	if !ok {
		inflightMap = map[string]interface{}{}
	}
	queueStats.MessagesInFlight = len(inflightMap)

	// Queue size stats
	ids, err := q.store.Get(fmt.Sprintf("queue:%s:_queue", q.name))
	if err != nil {
		return queueStats, errors.Wrap(err, "queue get")
	}
	idsSlice, ok := ids.([]interface{})
	if !ok {
		idsSlice = []interface{}{}
	}

	queueStats.MessagesInQueue = len(idsSlice)

	return queueStats, nil
}

func (q *LocalQueue) cleaner() {
cleanerLoop:
	for {
		select {
		case <-time.After(q.inflightTimeout / 2):
			if err := q.clean(); err != nil {
				log.Error(err)
			}
		case <-q.done:
			break cleanerLoop
		}
	}
}

func (q *LocalQueue) Close() {
	close(q.done)
}

func (q *LocalQueue) clean() error {
	now := time.Now()
	inflightKey := fmt.Sprintf("queue:%s:_inflight", q.name)
	q.inFlightLock.Lock()
	defer q.inFlightLock.Unlock()

	// Fetch all inflight messages
	inflightMessages, err := q.store.Get(inflightKey)
	if err != nil {
		return errors.Wrap(err, "get inflight")
	}
	inflightMap, ok := inflightMessages.(map[string]interface{})
	if !ok {
		inflightMap = map[string]interface{}{}
	}
	anyChanges := false
	for messageId, expiryInterface := range inflightMap {
		expiryString, ok := expiryInterface.(string)
		if !ok {
			log.Error("Expiry date not a string", expiryInterface)
			continue
		}
		time, err := time.Parse(time.RFC3339, expiryString)
		if err != nil {
			return errors.Wrap(err, "parse expriy date")
		}
		if now.After(time) {
			// Expired!
			message, err := q.getMessage(MessageID(messageId))
			if err != nil {
				return errors.Wrap(err, "lookup message")
			}
			message.Retries++
			if err := q.putMessage(message); err != nil {
				return errors.Wrap(err, "store updated message")
			}

			if err := q.queueMessage(message); err != nil {
				return errors.Wrap(err, "requeue message")
			}

			delete(inflightMap, messageId)

			anyChanges = true
		}
	}
	if anyChanges {
		// Put back inflight map
		if err := q.store.Put(inflightKey, inflightMap); err != nil {
			return errors.Wrap(err, "put inflight")
		}
	}

	return nil
}

func (q *LocalQueue) putMessage(message *Message) error {
	return q.store.Put(fmt.Sprintf("queue:%s:%s", q.name, message.ID), message)
}
