package emailcheck

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrMalformed    = errors.New("emailcheck: malformed address")
	ErrDisposable   = errors.New("emailcheck: disposable domain")
	ErrNoMailServer = errors.New("emailcheck: domain cannot receive mail")
	// Callers should let the address through, so a DNS outage never blocks real signups.
	ErrLookupUnavailable = errors.New("emailcheck: dns lookup unavailable")
)

type TypoError struct {
	// Full corrected address, e.g. "john@gmail.com", not just the domain.
	Suggested string
}

func (e *TypoError) Error() string {
	return "emailcheck: possible typo, did you mean " + e.Suggested
}

type Checker struct {
	disposable domainSet
	allow      domainSet
	providers  []string
	providerOf domainSet
	resolver   MXResolver
	mxTimeout  time.Duration
	mxEnabled  bool
}

// mxEnabled=false skips the DNS step so local development and CI work offline.
func New(resolver MXResolver, mxTimeout time.Duration, mxEnabled bool) (*Checker, error) {
	providers := parseDomainList(providerList)
	if len(providers) == 0 {
		return nil, errors.New("emailcheck: embedded provider list is empty")
	}
	disposable := newDomainSet(upstreamBlocklist, customBlocklist)
	if len(disposable) == 0 {
		return nil, errors.New("emailcheck: embedded disposable list is empty")
	}

	return &Checker{
		disposable: disposable,
		allow:      newDomainSet(customAllowlist),
		providers:  providers,
		providerOf: newDomainSet(providerList),
		resolver:   resolver,
		mxTimeout:  mxTimeout,
		mxEnabled:  mxEnabled,
	}, nil
}

// Typo runs before disposable because the upstream list marks squatted typo
// domains like gmial.com as disposable, and "did you mean gmail.com?" helps more.
func (c *Checker) Check(ctx context.Context, email string) error {
	local, domain, err := splitAddress(email)
	if err != nil {
		return err
	}

	// The allowlist exists to rescue real users blocked by mistake, so it
	// skips both list-based checks. The mail-server check still applies.
	if !c.allow.matches(domain) {
		if suggestion, ok := c.typoSuggestion(domain); ok {
			return &TypoError{Suggested: local + "@" + suggestion}
		}
		if c.disposable.matches(domain) {
			return ErrDisposable
		}
	}

	if !c.mxEnabled {
		return nil
	}
	return c.checkMailServer(ctx, domain)
}

// splitAddress accepts only a bare address. net/mail alone would also accept
// "Bob <bob@x.com>", which must not be stored as an email.
func splitAddress(email string) (local, domain string, err error) {
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Name != "" || parsed.Address != email {
		return "", "", ErrMalformed
	}

	at := strings.LastIndex(email, "@")
	local, domain = email[:at], email[at+1:]
	if local == "" || !strings.Contains(domain, ".") {
		return "", "", fmt.Errorf("%w: domain %q has no dot", ErrMalformed, domain)
	}
	return local, domain, nil
}
