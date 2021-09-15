package store

import (
	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
)

type LevelDBStore struct {
	path string
	db   *leveldb.DB
}

var _ Store = &LevelDBStore{}

func NewLevelDBStore(path string) (*LevelDBStore, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &LevelDBStore{
		path: path,
		db:   db,
	}, nil
}

func (s *LevelDBStore) Put(key string, val interface{}) error {
	jsonBuf, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return s.db.Put([]byte(key), jsonBuf, nil)
}

func (s *LevelDBStore) Get(key string) (interface{}, error) {
	valBuf, err := s.db.Get([]byte(key), nil)
	if err == leveldb.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err = json.Unmarshal(valBuf, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *LevelDBStore) Delete(key string) error {
	return s.db.Delete([]byte(key), nil)
}

type QueryResult struct {
	Key   string
	Value interface{}
}

func (s *LevelDBStore) loadIterator(iter iterator.Iterator) ([]QueryResult, error) {
	results := make([]QueryResult, 0, 20)
	for iter.Next() {
		var queryResult QueryResult
		queryResult.Key = string(iter.Key())
		if err := json.Unmarshal(iter.Value(), &queryResult.Value); err != nil {
			return nil, err
		}
		results = append(results, queryResult)
	}
	return results, nil
}

func (s *LevelDBStore) QueryRange(startKey string, endKey string) ([]QueryResult, error) {
	iter := s.db.NewIterator(&util.Range{Start: []byte(startKey), Limit: []byte(endKey)}, nil)
	return s.loadIterator(iter)
}

func (s *LevelDBStore) QueryPrefix(prefix string) ([]QueryResult, error) {
	iter := s.db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	return s.loadIterator(iter)
}

func (s *LevelDBStore) Close() error {
	return s.db.Close()
}

func (s *LevelDBStore) DeleteStore() error {
	s.Close()
	return os.RemoveAll(s.path)
}
