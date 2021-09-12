package ttl

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key interface{}) (interface{}, bool)

	Set(ctx context.Context, key, value interface{}) error

	// SetTTL allows for overriding the default timeout
	// for the cache for this value
	SetTTL(ctx context.Context, key, value interface{}, timeout *time.Duration) error

	Delete(ctx context.Context, key interface{})
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

	go func() {
		<-ctx.Done()
		c.cleanup()
	}()

	return c
}
