package queue

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/eventbus"
)

type MessageID string

type Message struct {
	ID      MessageID   `json:"id"`
	Body    interface{} `json:"body"`
	Retries int
}

type Stats struct {
	MessagesInFlight int
	MessagesInQueue  int
}

var NoMessageError error = errors.New("no message queued")

func GenerateMessageID() MessageID {
	return MessageID(uuid.NewString())
}

type Queue interface {
	Send(message *Message) error
	// Return nil, nil if no message waiting
	Receive() (*Message, error)
	Ack(id MessageID) error
	// exposes "message" event
	EventBus() eventbus.EventBus

	Stats() (Stats, error)

	Close()
}
