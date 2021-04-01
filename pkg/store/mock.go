package store

type MockStore struct {
}

func (m MockStore) Put(key string, val interface{}) error {
	panic("implement me")
}

func (m MockStore) Get(key string) (interface{}, error) {
	return key, nil
}

func (m MockStore) Delete(key string) error {
	panic("implement me")
}

func (m MockStore) QueryRange(startKey string, endKey string) ([]QueryResult, error) {
	panic("implement me")
}

func (m MockStore) QueryPrefix(prefix string) ([]QueryResult, error) {
	panic("implement me")
}

func (m MockStore) Close() error {
	return nil
}

var _ Store = &MockStore{}
