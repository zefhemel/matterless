package cache

type Cache interface {
	Get(key string) interface{}
	Set(key string, val interface{})
}
