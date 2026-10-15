package emailcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// nullMXHost is the RFC 7505 marker a domain publishes to say it accepts no mail.
const nullMXHost = "."

// A domain's mail setup rarely changes, so an hour of reuse is safe. The size
// cap keeps a flood of made-up domains from growing memory without bound.
const (
	mxCacheTTL        = time.Hour
	mxCacheMaxDomains = 10_000
)

type MXResolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

// Providers from providers.txt are known to receive mail, so they skip DNS,
// which covers most signups.
func (c *Checker) checkMailServer(ctx context.Context, domain string) error {
	if _, ok := c.providerOf[domain]; ok {
		return nil
	}
	if cached, ok := c.mxCache.get(domain); ok {
		return cached.err
	}

	err := c.lookupMailServer(ctx, domain)
	// Lookup failures are not cached, so a DNS blip does not stick for an hour.
	if err == nil || errors.Is(err, ErrNoMailServer) {
		c.mxCache.put(domain, err)
	}
	return err
}

// Any DNS failure other than "not found" returns ErrLookupUnavailable so callers can fail open.
func (c *Checker) lookupMailServer(ctx context.Context, domain string) error {
	ctx, cancel := context.WithTimeout(ctx, c.mxTimeout)
	defer cancel()

	records, err := c.resolver.LookupMX(ctx, domain)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return ErrNoMailServer
		}
		return fmt.Errorf("%w: %w", ErrLookupUnavailable, err)
	}

	if len(records) == 0 || (len(records) == 1 && records[0].Host == nullMXHost) {
		return ErrNoMailServer
	}
	return nil
}

type mxCacheEntry struct {
	err       error
	expiresAt time.Time
}

type mxCache struct {
	mu      sync.Mutex
	entries map[string]mxCacheEntry
}

func newMXCache() *mxCache {
	return &mxCache{entries: map[string]mxCacheEntry{}}
}

func (m *mxCache) get(domain string) (mxCacheEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[domain]
	if !ok || time.Now().After(entry.expiresAt) {
		return mxCacheEntry{}, false
	}
	return entry, true
}

func (m *mxCache) put(domain string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if len(m.entries) >= mxCacheMaxDomains {
		for d, entry := range m.entries {
			if now.After(entry.expiresAt) {
				delete(m.entries, d)
			}
		}
	}
	// Still full of live entries: drop an arbitrary one rather than refuse new domains.
	if len(m.entries) >= mxCacheMaxDomains {
		for d := range m.entries {
			delete(m.entries, d)
			break
		}
	}
	m.entries[domain] = mxCacheEntry{err: err, expiresAt: now.Add(mxCacheTTL)}
}
