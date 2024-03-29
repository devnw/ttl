// Package ttl implements a TTL cache which can be used to store
// values a specified timeout period. The cache implementation
// supports extending the timeout of regularly read values as well
// as storing custom TTL timeouts for specific key/value pairs
// with the `SetTTL` method.
package ttl

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Cache implements a TTL Cache which can be used to store
// values values a specified timeout period. If the value
// is read before the timeout period has passed, and the
// extend flag is set to true, the timeout period will be
// extended by the timeout period. This ensures that the
// value is not removed if it is regularly accessed within
// the timeout period.
type Cache[K comparable, V any] struct {
	ctx      context.Context
	timeout  time.Duration
	extend   bool
	values   map[K]*rw[V]
	valuesMu sync.RWMutex
}

// NewCache creates a new TTL Cache using the a timeout
// for the default timeout of stored values and the extend
// value to determine if the cache lifetime of the set values
// should be extended upon read
func NewCache[K comparable, V any](
	ctx context.Context,
	timeout time.Duration,
	extend bool,
) *Cache[K, V] {
	if ctx == nil {
		ctx = context.Background()
	}

	c := &Cache[K, V]{
		ctx:     ctx,
		timeout: timeout,
		extend:  extend,
		values:  make(map[K]*rw[V]),
	}

	go func() {
		<-ctx.Done()
		c.cleanup()
	}()

	return c
}

// Delete removes the values associated with the
// passed key from the cache
func (c *Cache[K, V]) Delete(ctx context.Context, key K) {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	rw, ok := c.values[key]
	if !ok {
		return
	}

	// Cancel the context and delete the map entry
	// for this key
	rw.cancel()
	delete(c.values, key)
}

// Get accesses the value for the Key provide
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	if c.values == nil {
		return *new(V), false
	}

	c.valuesMu.RLock()
	rw, ok := c.values[key]
	c.valuesMu.RUnlock()

	// No stored value for this key
	if !ok {
		return *new(V), ok
	}

	select {
	case <-c.ctx.Done():
		return *new(V), false
	case v, ok := <-rw.read:
		if !ok {
			return *new(V), ok
		}

		return v, ok
	}
}

// Set sets the value for the key provided
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) error {
	return c.SetTTL(ctx, key, value, c.timeout)
}

// SetTTL allows for direct control over the TTL of a specific
// Key in the cache which is passed as timeout in parameter three.
// This timeout can be `nil` which will keep the value permanently
// in the cache without expiration until it's deleted
func (c *Cache[K, V]) SetTTL(
	ctx context.Context,
	key K, value V,
	timeout time.Duration,
) error {
	if c.values == nil {
		return fmt.Errorf("canceled cache instance")
	}

	// Pull the parent context if the passed context is nil
	if ctx == nil {
		ctx = c.ctx
	}

	c.valuesMu.RLock()
	rw, ok := c.values[key]
	c.valuesMu.RUnlock()

	// No stored value for this key
	if !ok {
		return c.write(key, c.set(key, value, timeout, c.extend))
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case rw.write <- newvalue[V]{
		v:       value,
		timeout: timeout,
	}:
	}

	return nil
}
