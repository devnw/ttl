package ttl

import (
	"context"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func cleanrw[V any]() *rw[V] {
	ctx, cancel := context.WithCancel(context.Background())

	return &rw[V]{
		ctx,
		cancel,
		make(<-chan V),
		make(chan<- newvalue[V]),
	}
}

func Test_cache_write(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testdata := map[string]struct {
		key   string
		value *rw[any]
	}{
		"valid": {
			"test-key",
			cleanrw[any](),
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			c := NewCache[string, any](ctx, time.Hour, false)

			if c == nil {
				t.Fatal("Expected valid struct, got NIL")
			}

			err := c.write(test.key, test.value)
			if err != nil {
				t.Fatalf("expected success | %s", err.Error())
			}

			v, ok := c.values[test.key]
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

	c := NewCache[string, any](ctx, time.Hour, false)

	rw := cleanrw[any]()

	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	c.cleanup()

	err := c.write("test", rw)
	if err == nil {
		t.Fatal("expected failure due to nil value map")
	}
}

func Test_cache_write_invalid_value(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache[string, any](ctx, time.Hour, false)
	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	err := c.write("test", nil)
	if err == nil {
		t.Fatal("expected failure due to nil value")
	}
}

func Test_cache_cleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache[string, any](ctx, time.Hour, false)

	rw := cleanrw[any]()

	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	err := c.write("test", rw)
	if err != nil {
		t.Fatalf("expected success | %s", err.Error())
	}

	c.cleanup()

	// Ensure the stored value's context is canceled
	select {
	case <-time.Tick(time.Minute):
		t.Fatal("expected canceled context on value")
	case <-rw.ctx.Done():
	}

	if c.values != nil {
		t.Fatalf("cache not properly cleaned up")
	}
}

func Test_cache_set(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache[string, string](ctx, time.Hour, false)
	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	timeout := time.Minute

	rw := c.set("test", "test", timeout, false)

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
	case rw.write <- newvalue[string]{"test2", timeout}:
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

	c := NewCache[string, string](ctx, time.Hour, false)
	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	timeout := time.Minute

	rw := c.set("test", "test", timeout, false)

	if rw.cancel == nil ||
		rw.ctx == nil ||
		rw.read == nil ||
		rw.write == nil {
		t.Fatalf("invalid rw instantiation | %s", spew.Sdump(rw))
	}

	err := c.write("test", rw)
	if err != nil {
		t.Fatalf("error writing rw to cache | %s", err.Error())
	}

	if _, ok := c.values["test"]; !ok {
		t.Fatalf("value not in cache")
	}

	// Close write channel to shut down the rwloop
	close(rw.write)

	<-time.Tick(time.Second)

	if _, ok := c.values["test"]; !ok {
		t.Fatalf("value not in cache")
	}
}

func Test_cache_set_closedRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	che := &Cache[string, string]{
		ctx:     ctx,
		timeout: time.Hour,
		extend:  false,
		values:  make(map[string]*rw[string]),
	}

	if che == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	timeout := time.Minute

	outgoing := make(chan string)
	incoming := make(chan newvalue[string])

	out := &rw[string]{
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
