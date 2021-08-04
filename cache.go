package ttl

import (
	"context"
	"sync"
	"time"
)

type rw struct {
	ctx   context.Context
	read  <-chan interface{}
	write chan<- interface{}
}

type cache struct {
	ctx      context.Context
	timeout  time.Duration
	extend   bool
	values   map[interface{}]*rw
	valuesMu sync.RWMutex
}

// Delete removes the values associated with the
// passed key from the cache
func (c *cache) Delete(key interface{}) {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	rw, ok := c.values[key]
	if !ok {
		return
	}

	// Delete the map entry and close the write
	// channel so that it properly closes the routine
	defer close(rw.write)
	delete(c.values, key)
}

func (c *cache) Get(key interface{}) interface{} {
	c.valuesMu.RLock()
	rw, ok := c.values[key]
	c.valuesMu.RUnlock()

	// No stored value for this key
	if !ok {
		return nil
	}

	// TODO: This should have a timeout to ensure that
	// if there is a block on the read that the cache doesn't
	// create a deadlock
	select {
	case <-c.ctx.Done():
		return nil
	case v, ok := <-rw.read:
		if !ok {
			return nil
		}

		return v
	}
}

func (c *cache) Set(key, value interface{}) {

}

func (c *cache) SetTTL(key, value interface{}, timeout time.Duration) {

}

func (c *cache) set(
	ctx context.Context,
	key, value interface{},
	timeout time.Duration,
	extend bool,
) *rw {
	outgoing := make(chan interface{})
	incoming := make(chan interface{})
	out := &rw{
		read:  outgoing,
		write: incoming,
	}

	go func(
		key, value interface{},
		outgoing chan<- interface{},
		incoming <-chan interface{},
		timeout time.Duration,
		extend bool,
	) {
		defer c.Delete(key) // Cleanup the map entry
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

	}(key, value, outgoing, incoming, timeout, extend)

	return out
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
