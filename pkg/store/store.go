package store

type Store interface {
	Put(key string, val interface{}) error
	Get(key string) (interface{}, error) // Return nil, nil when not found
	Delete(key string) error
	QueryRange(startKey string, endKey string) ([]QueryResult, error)
	QueryPrefix(prefix string) ([]QueryResult, error)
	Close() error
	DeleteStore() error
}
