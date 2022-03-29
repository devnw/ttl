package ttl

import (
	"context"
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/davecgh/go-spew/spew"
)

const key = "test"
const value = "testvalue"

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
			c := NewCache[int, any](test.Context, test.timeout, test.extend)
			if c == nil {
				t.Fatal("Expected valid struct, got NIL")
			}

			if test.Context != nil && c.ctx != test.Context {
				t.Fatal("Expected context to match")
			} else if test.Context == nil && c.ctx == nil {
				t.Fatal("Expected non-nil context")
			}

			if c.timeout != test.timeout {
				t.Fatalf(
					"Expected timeout %s; got %s",
					test.timeout,
					c.timeout,
				)
			}

			if c.extend != test.extend {
				t.Fatalf(
					"Expected extend value %v; got %v",
					test.extend,
					c.extend,
				)
			}

			if c.values == nil {
				t.Fatal("Expected non-nil values map")
			}
		})
	}
}

func Test_Delete(t *testing.T) {
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

	c.Delete(ctx, "test")

	if _, ok := c.values["test"]; ok {
		t.Fatalf("value still in cache")
	}
	<-rw.ctx.Done()
}

func Test_Get(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCache[string, any](ctx, time.Hour, false)

	testdata := map[string]struct {
		key   string
		value any
	}{
		"nil": {
			"test",
			nil,
		},
		"string": {
			"teststring",
			"test1",
		},
		"int": {
			"testint", 10,
		},
		"float": {
			"testfloat", 1.25,
		},
		"bool": {
			"testbool", true,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			err := c.Set(ctx, test.key, test.value)
			if err != nil {
				t.Fatalf("error setting value in cache | %s", err)
			}

			v, ok := c.Get(ctx, test.key)
			if !ok {
				t.Fatalf("key [%s] missing from cache", test.key)
			}

			if v != test.value {
				t.Fatalf(
					"type mismatch; got %s; expected %s",
					spew.Sdump(v),
					spew.Sdump(test.value),
				)
			}
		})
	}
}

func Test_Get_nilmap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache[string, any](ctx, time.Hour, false)
	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	c.values = nil

	v, ok := c.Get(ctx, "test")
	if ok || v != nil {
		t.Fatal("expected failure to load from cache")
	}
}

func Test_Get_novalue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache[string, any](ctx, time.Hour, false)

	v, ok := c.Get(ctx, "test")
	if ok || v != nil {
		t.Fatal("expected failure to load from cache")
	}
}

func Test_Get_closedctx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	c := NewCache[string, any](ctx, time.Hour, false)
	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	cancel()

	// This will create a fake struct which has a channel
	// that will always block on read
	c.values["test"] = &rw[any]{}

	v, ok := c.Get(ctx, "test")
	if ok || v != nil {
		t.Fatal("expected failure to load from cache")
	}
}

func Test_Set(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCache[string, string](ctx, time.Hour, false)

	for i := 0; i < 1000; i++ {
		value := randomdata.SillyName()
		c.Set(ctx, key, value)

		v, ok := c.Get(ctx, key)
		if !ok {
			t.Fatal("expected key to exist")
		}

		if v != value {
			t.Fatalf("expected value %s; got %s", value, v)
		}
	}
}

func Test_Set_nilmap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache[string, any](ctx, time.Hour, false)
	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	c.values = nil

	err := c.Set(ctx, "test", "value")
	if err == nil {
		t.Fatal("expected error")
	}
}

func Test_Set_existing_update(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewCache[string, string](ctx, time.Hour, false)

	c.Set(ctx, key, value)

	testvalue := value
	for i := 0; i < 1000; i++ {
		v, ok := c.Get(ctx, "test")
		if !ok {
			t.Fatal("expected to find value")
		}

		if v != testvalue {
			t.Fatalf("expected [%s]; got [%s]", testvalue, v)
		}

		testvalue = randomdata.SillyName()
		err := c.Set(ctx, "test", testvalue)
		if err != nil {
			t.Fatal("error setting value")
		}
	}
}

func Test_Set_closedctx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx2, cancel2 := context.WithCancel(context.Background())

	c := NewCache[string, any](ctx, time.Hour, false)
	if c == nil {
		t.Fatal("Expected valid struct, got NIL")
	}

	// This will create a fake struct which has a channel
	// that will always block on read
	c.values["test"] = &rw[any]{}

	cancel2()
	err := c.Set(ctx2, "test", "value")
	if err == nil {
		t.Fatal("expected error")
	}
}

func Test_SetTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCache[string, string](ctx, time.Hour, false)

	for i := 0; i < 1000; i++ {
		value := randomdata.SillyName()

		timeout := time.Minute

		c.SetTTL(ctx, key, value, timeout)

		v, ok := c.Get(ctx, key)
		if !ok {
			t.Fatal("expected key to exist")
		}

		if v != value {
			t.Fatalf("expected value %s; got %s", value, v)
		}
	}
}

func Test_Set_expiration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCache[string, string](ctx, time.Hour, false)

	testdata := map[string]struct {
		expected  time.Duration
		tolerance time.Duration
	}{
		"1ms": {
			time.Millisecond,
			time.Microsecond * 500,
		},
		"10ms": {
			time.Millisecond * 10,
			time.Millisecond,
		},
		"50ms": {
			time.Millisecond * 50,
			time.Millisecond,
		},
		"500ms": {
			time.Millisecond * 500,
			time.Millisecond,
		},
		"1s": {
			time.Second,
			time.Millisecond * 50,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			err := c.SetTTL(ctx, key, name, test.expected)
			if err != nil {
				t.Fatal("expected successful set")
			}

			<-time.Tick(test.expected + test.tolerance)

			_, ok := c.Get(ctx, key)
			if ok {
				t.Fatal("expected expiration")
			}
		})
	}
}

func Test_Set_extension(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCache[string, string](ctx, time.Hour, true)

	testdata := map[string]struct {
		expected  time.Duration
		extension time.Duration
		tolerance time.Duration
	}{
		"500ms -> 3s": {
			time.Millisecond * 500,
			time.Second * 3,
			time.Millisecond * 50,
		},
		"1s -> 5s": {
			time.Second,
			time.Second * 5,
			time.Millisecond * 50,
		},
	}

	for name, test := range testdata {
		t.Run(name, func(t *testing.T) {
			err := c.SetTTL(ctx, key, name, test.expected)
			if err != nil {
				t.Fatal("expected successful set")
			}

			err = c.SetTTL(ctx, key, name, test.extension)
			if err != nil {
				t.Fatal("expected successful set")
			}

			<-time.Tick(test.extension - (test.expected - test.tolerance))

			_, ok := c.Get(ctx, key)
			if !ok {
				t.Fatal("expected extension")
			}

			<-time.Tick(test.extension + test.tolerance)

			_, ok = c.Get(ctx, key)
			if ok {
				t.Fatal("expected expiration")
			}
		})
	}
}

func Benchmark_Set(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCache[string, string](ctx, time.Hour, false)

	for n := 0; n < b.N; n++ {
		err := c.Set(ctx, key, value)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}

func Benchmark_Get(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewCache[string, string](ctx, time.Hour, false)
	c.Set(ctx, key, value)

	for n := 0; n < b.N; n++ {
		_, ok := c.Get(ctx, key)
		if !ok {
			b.Fatal("Key not found")
		}
	}
}
