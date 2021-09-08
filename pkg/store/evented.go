package store

type EventedStore struct {
	wrappedStore    Store
	putCallback     func(key string, val interface{})
	deletedCallback func(key string)
}

var _ Store = &EventedStore{}

func NewEventedStore(wrappedStore Store, putCallback func(key string, val interface{}), deletedCallback func(key string)) *EventedStore {
	return &EventedStore{
		wrappedStore:    wrappedStore,
		putCallback:     putCallback,
		deletedCallback: deletedCallback,
	}
}

func (s *EventedStore) Put(key string, val interface{}) error {
	if err := s.wrappedStore.Put(key, val); err != nil {
		return err
	} else {
		s.putCallback(key, val)
		return nil
	}
}

func (s *EventedStore) Delete(key string) error {
	if err := s.wrappedStore.Delete(key); err != nil {
		return err
	} else {
		s.deletedCallback(key)
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
