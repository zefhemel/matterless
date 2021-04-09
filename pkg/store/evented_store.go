package store

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/eventbus"
)

type EventedStore struct {
	wrappedStore Store
	eventBus     eventbus.EventBus
}

var _ Store = &EventedStore{}

func NewEventedStore(wrappedStore Store, eventBus eventbus.EventBus) *EventedStore {
	return &EventedStore{
		wrappedStore: wrappedStore,
		eventBus:     eventBus,
	}
}

type PutEvent struct {
	Key      string      `json:"key"`
	NewValue interface{} `json:"new_value"`
}

type DeleteEvent struct {
	Key string `json:"key"`
}

func (s *EventedStore) Put(key string, val interface{}) error {
	if err := s.wrappedStore.Put(key, val); err != nil {
		return err
	} else {
		s.eventBus.PublishAsync(fmt.Sprintf("store:put:%s", key), &PutEvent{
			Key:      key,
			NewValue: val,
		})
		return nil
	}
}

func (s *EventedStore) Delete(key string) error {
	if err := s.wrappedStore.Delete(key); err != nil {
		return err
	} else {
		s.eventBus.PublishAsync(fmt.Sprintf("store:del:%s", key), &DeleteEvent{
			Key: key,
		})
		return nil
	}
}

func (s *EventedStore) Get(key string) (interface{}, error) {
	return s.wrappedStore.Get(key)
}

func (s *EventedStore) QueryRange(startKey string, endKey string) ([]QueryResult, error) {
	return s.wrappedStore.QueryRange(startKey, endKey)
}

func (s *EventedStore) QueryPrefix(prefix string) ([]QueryResult, error) {
	return s.wrappedStore.QueryPrefix(prefix)
}

func (s *EventedStore) Close() error {
	return s.wrappedStore.Close()
}
