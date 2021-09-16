package ttl

import (
	"context"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func cleanrw() *rw {
	ctx, cancel := context.WithCancel(context.Background())

	return &rw{
		ctx,
		cancel,
		make(<-chan interface{}),
		make(chan<- newvalue),
	}
}

func Test_cache_write(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testdata := map[string]struct {
		key   interface{}
		value *rw
	}{
		"valid": {
			"test-key",
			cleanrw(),
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			c := NewCache(ctx, time.Hour, false)

			che, ok := c.(*cache)
			if !ok {
				t.Fatal("Invalid internal cache type")
			}

			if che == nil {
				t.Fatal("Expected valid struct, got NIL")
			}

			err := che.write(test.key, test.value)
			if err != nil {
				t.Fatalf("expected success | %s", err.Error())
			}

			v, ok := che.values[test.key]
			if !ok {
				t.Fatal("Key not in map")
			}

			if v != test.value {
				t.Fatal("Value does not match specified value")
			}
		})
	}
}

func Test_cache_write_cleaned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache(ctx, time.Hour, false)

	rw := cleanrw()

	che, ok := c.(*cache)
	if !ok {
		t.Fatal("Invalid internal cache type")
	}

	if che == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	che.cleanup()

	err := che.write("test", rw)
	if err == nil {
		t.Fatal("expected failure due to nil value map")
	}
}

func Test_cache_write_invalid_value(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache(ctx, time.Hour, false)

	che, ok := c.(*cache)
	if !ok {
		t.Fatal("Invalid internal cache type")
	}

	if che == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	err := che.write("test", nil)
	if err == nil {
		t.Fatal("expected failure due to nil value")
	}
}

func Test_cache_cleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache(ctx, time.Hour, false)

	rw := cleanrw()

	che, ok := c.(*cache)
	if !ok {
		t.Fatal("Invalid internal cache type")
	}

	if che == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	err := che.write("test", rw)
	if err != nil {
		t.Fatalf("expected success | %s", err.Error())
	}

	che.cleanup()

	// Ensure the stored value's context is canceled
	select {
	case <-time.Tick(time.Minute):
		t.Fatal("expected canceled context on value")
	case <-rw.ctx.Done():
	}

	if che.values != nil {
		t.Fatalf("cache not properly cleaned up")
	}
}

func Test_cache_set(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache(ctx, time.Hour, false)

	che, ok := c.(*cache)
	if !ok {
		t.Fatal("Invalid internal cache type")
	}

	if che == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	timeout := time.Minute

	rw := che.set("test", "test", timeout, false)

	if rw.cancel == nil ||
		rw.ctx == nil ||
		rw.read == nil ||
		rw.write == nil {
		t.Fatalf("invalid rw instantiation | %s", spew.Sdump(rw))
	}

	select {
	case <-ctx.Done():
		t.Fatal("expected read")
	case v, ok := <-rw.read:
		if !ok {
			t.Fatal("closed read channel")
		}

		if v != "test" {
			t.Fatalf("invalid internal value; expected 'test' got %s", v)
		}
	}

	select {
	case <-ctx.Done():
		t.Fatal("expected write")
	case rw.write <- newvalue{"test2", timeout}:
	}

	select {
	case <-ctx.Done():
		t.Fatal("expected read")
	case v, ok := <-rw.read:
		if !ok {
			t.Fatal("closed read channel")
		}

		if v != "test2" {
			t.Fatalf("invalid internal value; expected 'test2' got %s", v)
		}
	}
}

func Test_cache_set_closedWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache(ctx, time.Hour, false)

	che, ok := c.(*cache)
	if !ok {
		t.Fatal("Invalid internal cache type")
	}

	if che == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	timeout := time.Minute

	rw := che.set("test", "test", timeout, false)

	if rw.cancel == nil ||
		rw.ctx == nil ||
		rw.read == nil ||
		rw.write == nil {
		t.Fatalf("invalid rw instantiation | %s", spew.Sdump(rw))
	}

	err := che.write("test", rw)
	if err != nil {
		t.Fatalf("error writing rw to cache | %s", err.Error())
	}

	if _, ok := che.values["test"]; !ok {
		t.Fatalf("value not in cache")
	}

	// Close write channel to shut down the rwloop
	close(rw.write)

	<-time.Tick(time.Second)

	if _, ok := che.values["test"]; !ok {
		t.Fatalf("value not in cache")
	}
}

func Test_cache_set_closedRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	che := &cache{
		ctx:     ctx,
		timeout: time.Hour,
		extend:  false,
		values:  make(map[interface{}]*rw),
	}

	if che == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	timeout := time.Minute

	outgoing := make(chan interface{})
	incoming := make(chan newvalue)

	out := &rw{
		ctx:    ctx,
		cancel: cancel,
		read:   outgoing,
		write:  incoming,
	}

	// Close write channel to shut down the rwloop
	close(outgoing)

	err := che.write("test", out)
	if err != nil {
		t.Fatalf("error writing rw to cache | %s", err.Error())
	}

	if _, ok := che.values["test"]; !ok {
		t.Fatalf("value not in cache")
	}

	che.rwloop(
		ctx,
		"test",
		"test",
		outgoing,
		incoming,
		timeout,
		false,
	)

	if _, ok := che.values["test"]; ok {
		t.Fatalf("value still in cache")
	}
}
