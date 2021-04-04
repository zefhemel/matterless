package queue_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/queue"
	"github.com/zefhemel/matterless/pkg/store"
	"os"
	"testing"
	"time"
)

func TestLocalQueue(t *testing.T) {
	s, err := store.NewLevelDBStore("test_store")
	defer os.RemoveAll("test_store")
	assert.NoError(t, err)

	q := queue.NewLocalQueue(s, "test", 2*time.Second)
	defer q.Close()
	assert.NoError(t, q.Send(&queue.Message{
		ID:   "123",
		Body: "My message",
	}))

	stats, err := q.Stats()
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.MessagesInQueue)
	assert.Equal(t, 0, stats.MessagesInFlight)

	msg, err := q.Receive()
	assert.NoError(t, err)
	assert.Equal(t, queue.MessageID("123"), msg.ID)
	assert.Equal(t, "My message", msg.Body)

	stats, err = q.Stats()
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.MessagesInQueue)
	assert.Equal(t, 1, stats.MessagesInFlight)

	assert.NoError(t, q.Ack(msg.ID))

	stats, err = q.Stats()
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.MessagesInQueue)
	assert.Equal(t, 0, stats.MessagesInFlight)

	// Ok, now send a bunch of messages
	for i := 0; i < 100; i++ {
		assert.NoError(t, q.Send(&queue.Message{
			ID:   queue.GenerateMessageID(),
			Body: float64(i),
		}))
	}
	stats, err = q.Stats()
	assert.NoError(t, err)
	assert.Equal(t, 100, stats.MessagesInQueue)
	assert.Equal(t, 0, stats.MessagesInFlight)

	ids := []queue.MessageID{}
	for i := 0; i < 100; i++ {
		msg, err := q.Receive()
		assert.NoError(t, err)
		assert.Equal(t, float64(i), msg.Body)
		ids = append(ids, msg.ID)
	}

	_, err = q.Receive()
	assert.Equal(t, queue.NoMessageError, err)

	stats, err = q.Stats()
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.MessagesInQueue)
	assert.Equal(t, 100, stats.MessagesInFlight)

	for _, messageID := range ids {
		assert.NoError(t, q.Ack(messageID))
	}

	stats, err = q.Stats()
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.MessagesInQueue)
	assert.Equal(t, 0, stats.MessagesInFlight)

	// Test expiry
	assert.NoError(t, q.Send(&queue.Message{
		ID:   "123",
		Body: "My message",
	}))

	msg, err = q.Receive()
	assert.NoError(t, err)
	assert.Equal(t, queue.MessageID("123"), msg.ID)

	// Wait
	time.Sleep(3 * time.Second)

	msg, err = q.Receive()
	assert.NoError(t, err)
	assert.Equal(t, queue.MessageID("123"), msg.ID)
	assert.Equal(t, 1, msg.Retries)

}
