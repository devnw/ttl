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

func (c *cache) write(key interface{}, value *rw) error {
	if c.values == nil || value == nil {
		return fmt.Errorf("invalid cache instance")
	}

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	c.values[key] = value

	return nil
}

func (c *cache) cleanup() {
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

		select {
		case <-ctx.Done():
			return
		default:
			c.Delete(ctx, key) // Cleanup the map entry
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

			value = v.v

			resetTimer(t, v.timeout)
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
