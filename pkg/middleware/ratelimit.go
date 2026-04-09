// Package middleware — ratelimit.go.
// RateLimit enforces a per-IP request limit on sensitive endpoints.
// Currently uses an in-process sliding window counter backed by a sync.Map.
// For multi-instance deployments, replace the store with Redis.
package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fluentfox/api/pkg/response"
)

// windowEntry tracks the request count and window start for one IP.
type windowEntry struct {
	mu        sync.Mutex
	count     int
	windowEnd time.Time
}

// RateLimiter holds per-IP state.
type RateLimiter struct {
	store        sync.Map
	maxRequests  int
	windowPeriod time.Duration
}

// NewRateLimiter constructs a RateLimiter.
// maxRequests is the allowed count per windowPeriod per IP.
func NewRateLimiter(maxRequests int, windowPeriod time.Duration) *RateLimiter {
	rl := &RateLimiter{
		maxRequests:  maxRequests,
		windowPeriod: windowPeriod,
	}
	go rl.cleanup()
	return rl
}

// Limit returns a chi middleware that enforces the rate limit.
// Returns HTTP 429 with a Retry-After header when the limit is exceeded.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)

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
			response.TooManyRequests(w, int(rl.windowPeriod.Seconds()))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cleanup periodically removes expired window entries to prevent unbounded growth.
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

// extractIP returns the client IP from X-Forwarded-For or RemoteAddr.
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	addr := r.RemoteAddr
	if colon := strings.LastIndex(addr, ":"); colon != -1 {
		return addr[:colon]
	}
	return addr
}
