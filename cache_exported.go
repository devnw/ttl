package ttl

import "time"

type Cache interface {
	Get(key interface{}) interface{}
	Set(key, value interface{})
	Delete(key interface{})
}

type TTLCache interface {
	Cache

	// SetTTL allows for overriding the default timeout
	// for the cache for this value
	SetTTL(key, value interface{}, timeout time.Duration)
}

// NewCache creates a new TTL Cache using the a timeout
// for the default timeout of stored values and the extend
// value to determine if the cache lifetime of the set values
// should be extended upon read
func NewCache(timeout time.Duration, extend bool) TTLCache {
	return nil
}
