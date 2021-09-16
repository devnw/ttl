package ttl

import (
	"context"
	"fmt"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key interface{}) (interface{}, bool)

	Set(ctx context.Context, key, value interface{}) error

	// SetTTL allows for overriding the default timeout
	// for the cache for this value
	SetTTL(ctx context.Context, key, value interface{}, timeout time.Duration) error

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

// Delete removes the values associated with the
// passed key from the cache
func (c *cache) Delete(
	ctx context.Context,
	key interface{},
) {
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

func (c *cache) Get(
	ctx context.Context,
	key interface{},
) (interface{}, bool) {
	if c.values == nil {
		return nil, false
	}

	c.valuesMu.RLock()
	rw, ok := c.values[key]
	c.valuesMu.RUnlock()

	// No stored value for this key
	if !ok {
		return nil, ok
	}

	select {
	case <-c.ctx.Done():
		return nil, false
	case v, ok := <-rw.read:
		if !ok {
			return nil, ok
		}

		return v, ok
	}
}

func (c *cache) Set(
	ctx context.Context,
	key, value interface{},
) error {
	return c.SetTTL(ctx, key, value, c.timeout)
}

// SetTTL allows for direct control over the TTL of a specific
// Key in the cache which is passed as timeout in parameter three.
// This timeout can be `nil` which will keep the value permanently
// in the cache without expiration until it's deleted
func (c *cache) SetTTL(
	ctx context.Context,
	key, value interface{},
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
	case rw.write <- newvalue{
		v:       value,
		timeout: timeout,
	}:
	}

	return nil
}
