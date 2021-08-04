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
	ctx        context.Context
	defaultTTL time.Duration
	values     map[interface{}]rw
	valuesMu   sync.RWMutex
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
