package store

type EmptyStore struct {
}

func (m EmptyStore) DeleteStore() error {
	return nil
}

func (m EmptyStore) Put(key string, val interface{}) error {
	return nil
}

func (m EmptyStore) Get(key string) (interface{}, error) {
	return nil, nil
}

func (m EmptyStore) Delete(key string) error {
	return nil
}

func (m EmptyStore) QueryRange(startKey string, endKey string) ([]QueryResult, error) {
	panic("implement me")
}

func (m EmptyStore) QueryPrefix(prefix string) ([]QueryResult, error) {
	panic("implement me")
}

func (m EmptyStore) Close() error {
	return nil
}

var _ Store = &EmptyStore{}
