package cache

import (
	"encoding/json"
	"os"
)

type OnDiskCache struct {
	path string
	data map[string]interface{}
}

func (c *OnDiskCache) Get(key string) interface{} {
	return c.data[key]
}

func (c *OnDiskCache) Set(key string, value interface{}) {
	c.data[key] = value
	c.Flush()
}

func (c *OnDiskCache) Flush() error {
	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer f.Close()
	jsonEncoder := json.NewEncoder(f)
	err = jsonEncoder.Encode(c.data)
	if err != nil {
		return err
	}
	return nil
}

func NewOnDiskCache(path string) *OnDiskCache {
	c := &OnDiskCache{
		path: path,
		data: map[string]interface{}{},
	}

	f, err := os.Open(path)
	if err != nil {
		// Something up with the cache, skip
		return c
	}
	defer f.Close()

	jsonDecoder := json.NewDecoder(f)
	err = jsonDecoder.Decode(&c.data)
	if err != nil {
		// Something up with the cache, skip
		return c
	}
	return c
}

var _ Cache = &OnDiskCache{}
