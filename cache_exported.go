package ttl

type Cache interface {
	Get(key interface{}) interface{}
	Set(key, value interface{})
}
