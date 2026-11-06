package cache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// TTL is a small in-memory read-through cache for values that change rarely
// (course content). Each key is loaded at most once per TTL: concurrent misses
// for the same key share a single load (singleflight), so a burst of requests
// after a deploy costs one set of queries, not one per request. Errors are
// never cached. Values are shared between callers and must be treated as
// read-only.
type TTL[V any] struct {
	ttl        time.Duration
	maxEntries int
	now        func() time.Time

	mu    sync.RWMutex
	items map[string]ttlItem[V]
	group singleflight.Group
}

type ttlItem[V any] struct {
	value   V
	expires time.Time
}

// NewTTL returns a cache that keeps each value for ttl. Once maxEntries keys
// are held, new keys are still served but not stored, so unbounded input
// (e.g. arbitrary path segments) cannot grow memory without limit.
func NewTTL[V any](ttl time.Duration, maxEntries int) *TTL[V] {
	return &TTL[V]{ttl: ttl, maxEntries: maxEntries, now: time.Now, items: make(map[string]ttlItem[V])}
}

// Get returns the cached value for key, or calls load and caches its result.
// load runs without the caller's cancellation, so one client hanging up does
// not fail the other requests waiting on the same load.
func (c *TTL[V]) Get(ctx context.Context, key string, load func(context.Context) (V, error)) (V, error) {
	if v, ok := c.lookup(key); ok {
		return v, nil
	}
	shared, err, _ := c.group.Do(key, func() (any, error) {
		if v, ok := c.lookup(key); ok {
			return v, nil
		}
		v, err := load(context.WithoutCancel(ctx))
		if err != nil {
			return v, err
		}
		c.store(key, v)
		return v, nil
	})
	if err != nil {
		var zero V
		return zero, err //nolint:wrapcheck // the caller's own load error; callers add their context
	}
	return shared.(V), nil //nolint:forcetypeassert // only V is ever stored in the group
}

func (c *TTL[V]) lookup(key string) (V, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || c.now().After(item.expires) {
		var zero V
		return zero, false
	}
	return item.value, true
}

func (c *TTL[V]) store(key string, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[key]; !exists && len(c.items) >= c.maxEntries {
		c.evictExpiredLocked()
		if len(c.items) >= c.maxEntries {
			return
		}
	}
	c.items[key] = ttlItem[V]{value: v, expires: c.now().Add(c.ttl)}
}

func (c *TTL[V]) evictExpiredLocked() {
	now := c.now()
	for k, item := range c.items {
		if now.After(item.expires) {
			delete(c.items, k)
		}
	}
}

// Len reports how many keys are held (expired ones included until evicted).
func (c *TTL[V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
