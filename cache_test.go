package ttl

import (
	"context"
	"testing"
	"time"
)

func Test_NewCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testdata := map[string]struct {
		context.Context
		timeout time.Duration
		extend  bool
	}{
		"valid - no extend": {
			ctx,
			time.Minute,
			false,
		},
		"valid - extend": {
			ctx,
			time.Minute,
			false,
		},
		"valid - nil context": {
			nil,
			time.Minute,
			false,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			c := NewCache(test.Context, test.timeout, test.extend)

			che, ok := c.(*cache)
			if !ok {
				t.Fatal("Invalid internal cache type")
			}

			if che == nil {
				t.Fatal("Expected valid struct, got NIL")
			}

			if test.Context != nil && che.ctx != test.Context {
				t.Fatal("Expected context to match")
			} else if test.Context == nil && che.ctx == nil {
				t.Fatal("Expected non-nil context")
			}

			if che.timeout != test.timeout {
				t.Fatalf(
					"Expected timeout %s; got %s",
					test.timeout,
					che.timeout,
				)
			}

			if che.extend != test.extend {
				t.Fatalf(
					"Expected extend value %v; got %v",
					test.extend,
					che.extend,
				)
			}

			if che.values == nil {
				t.Fatal("Expected non-nil values map")
			}
		})
	}
}

func cleanrw() *rw {
	ctx, cancel := context.WithCancel(context.Background())

	return &rw{
		ctx,
		cancel,
		make(<-chan interface{}),
		make(chan<- interface{}),
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

			che.write(test.key, test.value)

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
