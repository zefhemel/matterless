package cache

type InMemoryCache struct {
	data map[string]interface{}
}

func (c *InMemoryCache) Get(key string) interface{} {
	return c.data[key]
}

func (c *InMemoryCache) Set(key string, value interface{}) {
	c.data[key] = value
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data: map[string]interface{}{},
	}
}

var _ Cache = &InMemoryCache{}
