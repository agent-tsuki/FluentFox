package emailcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// nullMXHost is the RFC 7505 marker a domain publishes to say it accepts no mail.
const nullMXHost = "."

type MXResolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

// Any DNS failure other than "not found" returns ErrLookupUnavailable so callers can fail open.
func (c *Checker) checkMailServer(ctx context.Context, domain string) error {
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
