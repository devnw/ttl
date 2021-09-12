package ttl

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type newvalue struct {
	v       interface{}
	timeout *time.Duration
}

type rw struct {
	ctx    context.Context
	cancel context.CancelFunc
	read   <-chan interface{}
	write  chan<- newvalue
}

type cache struct {
	ctx      context.Context
	timeout  time.Duration
	extend   bool
	values   map[interface{}]*rw
	valuesMu sync.RWMutex
}

func (c *cache) write(key interface{}, value *rw) {
	if c.values == nil || value == nil {
		return
	}

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	c.values[key] = value
}

func (c *cache) cleanup() {
	<-c.ctx.Done()

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	// Cancel contexts
	for _, value := range c.values {
		value.cancel()
	}

	// Nil the map out so nothing can write
	// to the cache
	c.values = nil
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

	// TODO: This should have a timeout to ensure that
	// if there is a block on the read that the cache doesn't
	// create a deadlock
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
	return c.SetTTL(ctx, key, value, &c.timeout)
}

// SetTTL allows for direct control over the TTL of a specific
// Key in the cache which is passed as timeout in parameter three.
// This timeout can be `nil` which will keep the value permanently
// in the cache without expiration until it's deleted
func (c *cache) SetTTL(
	ctx context.Context,
	key, value interface{},
	timeout *time.Duration,
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
		c.write(key, c.set(key, value, timeout, c.extend))
		return nil
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

func (c *cache) set(
	key, value interface{},
	timeout *time.Duration,
	extend bool,
) *rw {
	ctx, cancel := context.WithCancel(c.ctx)
	outgoing := make(chan interface{})
	incoming := make(chan newvalue)

	out := &rw{
		ctx:    ctx,
		cancel: cancel,
		read:   outgoing,
		write:  incoming,
	}

	go c.rwloop(
		ctx,
		key,
		value,
		outgoing,
		incoming,
		timeout,
		extend,
	)

	return out
}

func (c *cache) rwloop(
	ctx context.Context,
	key, value interface{},
	outgoing chan<- interface{},
	incoming <-chan newvalue,
	timeout *time.Duration,
	extend bool,
) {
	defer func() {
		// Recover from any panic (most likely closed channel)
		// NOTE: This is ignored on purpose because the next
		// defer removes this key from the cache
		_ = recover()

		if timeout != nil {
			c.Delete(ctx, key) // Cleanup the map entry
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			// Re-initialize this map entry since this key/value is expected
			// to persist in the cache
			c.write(key, c.set(key, value, timeout, extend))
		}
	}()

	// Create the internal timer if the timeout is non-nil
	// and assign the internal `C` channel to the timer channel
	// for use in the select. Otherwise leave the timer channel
	// nil so that it never trips the select statement because
	// this specific key/value should be persistent.
	var t *time.Timer
	var timer <-chan time.Time
	if timeout != nil {
		t = time.NewTimer(*timeout)
		timer = t.C
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer:
			return
		case v, ok := <-incoming:
			if !ok {
				continue
			}

			value = v

			resetTimer(t, timeout)
		case outgoing <- value:
			// Only extend the timer on read
			// if it is configured to do so
			if !extend {
				continue
			}

			resetTimer(t, timeout)
		}
	}
}

// resetTimer resets the timer instance using the
// duration passed in. This uses the recommended
// set of calls from the go doc for `time.Timer.Reset`
// to ensure the the `C` channel is drained and doesn't
// immediately read on reset
func resetTimer(t *time.Timer, d *time.Duration) {
	if d == nil {
		return
	}

	if !t.Stop() {
		<-t.C
	}
	t.Reset(*d)
}
