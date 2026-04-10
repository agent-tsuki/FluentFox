// Package middleware — ratelimit.go
// Per-IP sliding window rate limiter backed by sync.Map.
// For multi-instance deployments, replace the store with Redis.
package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/fluentfox/api/pkg/response"
)

type windowEntry struct {
	mu        sync.Mutex
	count     int
	windowEnd time.Time
}

// RateLimiter holds per-IP window state.
type RateLimiter struct {
	store        sync.Map
	maxRequests  int
	windowPeriod time.Duration
}

// NewRateLimiter constructs a RateLimiter.
// maxRequests is the allowed count per windowPeriod per IP.
func NewRateLimiter(maxRequests int, windowPeriod time.Duration) *RateLimiter {
	rl := &RateLimiter{maxRequests: maxRequests, windowPeriod: windowPeriod}
	go rl.cleanup()
	return rl
}

// Limit returns a Gin middleware that enforces the rate limit.
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		raw, _ := rl.store.LoadOrStore(ip, &windowEntry{
			windowEnd: time.Now().Add(rl.windowPeriod),
		})
		entry := raw.(*windowEntry)

		entry.mu.Lock()
		now := time.Now()
		if now.After(entry.windowEnd) {
			entry.count = 0
			entry.windowEnd = now.Add(rl.windowPeriod)
		}
		entry.count++
		count := entry.count
		entry.mu.Unlock()

		if count > rl.maxRequests {
			response.TooManyRequests(c.Writer, int(rl.windowPeriod.Seconds()))
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.store.Range(func(key, value any) bool {
			entry := value.(*windowEntry)
			entry.mu.Lock()
			expired := now.After(entry.windowEnd)
			entry.mu.Unlock()
			if expired {
				rl.store.Delete(key)
			}
			return true
		})
	}
}
