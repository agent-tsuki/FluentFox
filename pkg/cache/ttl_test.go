package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTTLCachesUntilExpiry(t *testing.T) {
	now := time.Unix(0, 0)
	c := NewTTL[int](time.Minute, 10)
	c.now = func() time.Time { return now }
	loads := 0
	load := func(context.Context) (int, error) { loads++; return loads, nil }

	for range 3 {
		if v, err := c.Get(context.Background(), "k", load); err != nil || v != 1 {
			t.Fatalf("got %d, %v; want 1 from the first load", v, err)
		}
	}
	now = now.Add(time.Minute + time.Second)
	if v, _ := c.Get(context.Background(), "k", load); v != 2 {
		t.Fatalf("after expiry got %d, want a fresh load (2)", v)
	}
}

func TestTTLDoesNotCacheErrors(t *testing.T) {
	c := NewTTL[int](time.Minute, 10)
	fail := errors.New("db down")
	if _, err := c.Get(context.Background(), "k", func(context.Context) (int, error) { return 0, fail }); !errors.Is(err, fail) {
		t.Fatalf("got %v, want the load error", err)
	}
	if v, err := c.Get(context.Background(), "k", func(context.Context) (int, error) { return 7, nil }); err != nil || v != 7 {
		t.Fatalf("got %d, %v; want a retry after the error", v, err)
	}
}

func TestTTLSharesConcurrentLoads(t *testing.T) {
	c := NewTTL[int](time.Minute, 10)
	var loads atomic.Int32
	release := make(chan struct{})
	load := func(context.Context) (int, error) { //nolint:unparam // must match the loader signature
		loads.Add(1)
		<-release
		return 42, nil
	}

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if v, err := c.Get(context.Background(), "k", load); err != nil || v != 42 {
				t.Errorf("got %d, %v", v, err)
			}
		})
	}
	time.Sleep(20 * time.Millisecond) // let the goroutines pile up on the miss
	close(release)
	wg.Wait()
	if n := loads.Load(); n != 1 {
		t.Fatalf("%d loads for 50 concurrent misses, want 1", n)
	}
}

func TestTTLLoadIgnoresCallerCancellation(t *testing.T) {
	c := NewTTL[int](time.Minute, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v, err := c.Get(ctx, "k", func(ctx context.Context) (int, error) { return 1, ctx.Err() })
	if err != nil || v != 1 {
		t.Fatalf("got %d, %v; a cancelled caller must not cancel the shared load", v, err)
	}
}

func TestTTLCapsEntries(t *testing.T) {
	c := NewTTL[int](time.Minute, 2)
	for _, k := range []string{"a", "b", "c"} {
		if _, err := c.Get(context.Background(), k, func(context.Context) (int, error) { return 1, nil }); err != nil {
			t.Fatal(err)
		}
	}
	if n := c.Len(); n != 2 {
		t.Fatalf("holds %d keys, want the cap of 2", n)
	}
}
