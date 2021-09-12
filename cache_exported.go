package ttl

import (
	"context"
	"time"
)

type Cache interface {
	Get(key interface{}) (interface{}, bool)
	Set(key, value interface{})

	// SetTTL allows for overriding the default timeout
	// for the cache for this value
	SetTTL(key, value interface{}, timeout *time.Duration)

	Delete(key interface{})
}

// NewCache creates a new TTL Cache using the a timeout
// for the default timeout of stored values and the extend
// value to determine if the cache lifetime of the set values
// should be extended upon read
func NewCache(ctx context.Context, timeout time.Duration, extend bool) Cache {
	if ctx == nil {
		ctx = context.Background()
	}

	c := &cache{
		ctx:     ctx,
		timeout: timeout,
		extend:  extend,
		values:  make(map[interface{}]*rw),
	}

	defer c.cleanup()

	return c
}
