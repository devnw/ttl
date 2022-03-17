package ttl

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type newvalue[V any] struct {
	v       V
	timeout time.Duration
}

type rw[V any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	read   <-chan V
	write  chan<- newvalue[V]
}

type cache[K comparable, V any] struct {
	ctx      context.Context
	timeout  time.Duration
	extend   bool
	values   map[K]*rw[V]
	valuesMu sync.RWMutex
}

func (c *cache[K, V]) write(key K, value *rw[V]) error {
	if c.values == nil || value == nil {
		return fmt.Errorf("invalid cache instance")
	}

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	c.values[key] = value

	return nil
}

func (c *cache[K, V]) cleanup() {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	if c.values == nil {
		return
	}

	// Cancel contexts
	for _, value := range c.values {
		if value == nil || value.cancel == nil {
			continue
		}

		value.cancel()
	}

	// Nil the map out so nothing can write
	// to the cache
	c.values = nil
}

func (c *cache[K, V]) set(
	key K,
	value V,
	timeout time.Duration,
	extend bool,
) *rw[V] {
	ctx, cancel := context.WithCancel(c.ctx)
	outgoing := make(chan V)
	incoming := make(chan newvalue[V])

	out := &rw[V]{
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

func (c *cache[K, V]) rwloop(
	ctx context.Context,
	key K,
	value V,
	outgoing chan<- V,
	incoming <-chan newvalue[V],
	timeout time.Duration,
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

	defer close(outgoing)

	// Create the internal timer if the timeout is non-nil
	// and assign the internal `C` channel to the timer channel
	// for use in the select. Otherwise leave the timer channel
	// nil so that it never trips the select statement because
	// this specific key/value should be persistent.

	t := time.NewTimer(timeout)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			return
		case v, ok := <-incoming:
			if !ok {
				continue
			}

			value = v.v
			timeout = v.timeout

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
func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		<-t.C
	}
	t.Reset(d)
}
