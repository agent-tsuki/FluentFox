package emailcheck

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

type fakeResolver struct {
	fn    func(ctx context.Context, name string) ([]*net.MX, error)
	calls int
}

func (f *fakeResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	f.calls++
	return f.fn(ctx, name)
}

func okMX(context.Context, string) ([]*net.MX, error) {
	return []*net.MX{{Host: "mx.example.com.", Pref: 10}}, nil
}

func newChecker(t *testing.T, r *fakeResolver, mxEnabled bool) *Checker {
	t.Helper()
	c, err := New(r, time.Second, mxEnabled)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestChecker_Check(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		resolve       func(context.Context, string) ([]*net.MX, error)
		mxDisabled    bool
		wantErr       error
		wantSuggested string
		wantCalls     int
	}{
		{name: "flags gmial.com as typo of gmail.com", email: "a@gmial.com", wantSuggested: "a@gmail.com"},
		{name: "flags gmail.con as typo of gmail.com", email: "a@gmail.con", wantSuggested: "a@gmail.com"},
		{name: "flags yahooo.com as typo of yahoo.com", email: "a@yahooo.com", wantSuggested: "a@yahoo.com"},
		{name: "flags outlok.com as typo of outlook.com", email: "a@outlok.com", wantSuggested: "a@outlook.com"},
		{name: "does not flag yahoo.co.jp", email: "a@yahoo.co.jp", wantCalls: 1},
		{name: "does not flag hotmail.co.jp", email: "a@hotmail.co.jp", wantCalls: 1},
		{name: "does not flag yahoo.co.id", email: "a@yahoo.co.id", wantCalls: 1},
		{name: "does not flag mail.com", email: "a@mail.com", wantCalls: 1},
		{name: "does not flag email.com", email: "a@email.com", wantCalls: 1},
		{name: "does not flag proton.me", email: "a@proton.me", wantCalls: 1},
		{name: "does not flag hive.com", email: "a@hive.com", wantCalls: 1},
		{name: "rejects disposable yopmail.com", email: "a@yopmail.com", wantErr: ErrDisposable},
		{name: "rejects subdomain of disposable domain", email: "a@x.mailinator.com", wantErr: ErrDisposable},
		{name: "rejects display-name address as malformed", email: "Bob <a@b.com>", wantErr: ErrMalformed},
		{name: "rejects domain without dot as malformed", email: "a@localhost", wantErr: ErrMalformed},
		{name: "rejects empty input as malformed", email: "", wantErr: ErrMalformed},
		{
			name:  "returns ErrNoMailServer when domain not found",
			email: "a@nope-example.org", wantCalls: 1, wantErr: ErrNoMailServer,
			resolve: func(context.Context, string) ([]*net.MX, error) { return nil, &net.DNSError{IsNotFound: true} },
		},
		{
			name:  "returns ErrNoMailServer when no records",
			email: "a@nope-example.org", wantCalls: 1, wantErr: ErrNoMailServer,
			resolve: func(context.Context, string) ([]*net.MX, error) { return nil, nil },
		},
		{
			name:  "returns ErrNoMailServer for null MX",
			email: "a@nope-example.org", wantCalls: 1, wantErr: ErrNoMailServer,
			resolve: func(context.Context, string) ([]*net.MX, error) { return []*net.MX{{Host: "."}}, nil },
		},
		{
			name:  "returns ErrLookupUnavailable on DNS timeout",
			email: "a@nope-example.org", wantCalls: 1, wantErr: ErrLookupUnavailable,
			resolve: func(context.Context, string) ([]*net.MX, error) {
				return nil, &net.DNSError{IsTimeout: true}
			},
		},
		{name: "passes when a valid MX record exists", email: "a@example.org", wantCalls: 1},
		{name: "skips resolver when MX check is disabled", email: "a@example.org", mxDisabled: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn := tc.resolve
			if fn == nil {
				fn = okMX
			}
			r := &fakeResolver{fn: fn}
			c := newChecker(t, r, !tc.mxDisabled)

			err := c.Check(context.Background(), tc.email)

			if tc.wantSuggested != "" {
				var typo *TypoError
				if !errors.As(err, &typo) {
					t.Fatalf("want TypoError, got %v", err)
				}
				if typo.Suggested != tc.wantSuggested {
					t.Errorf("Suggested = %q, want %q", typo.Suggested, tc.wantSuggested)
				}
			} else if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("err = %v, want %v", err, tc.wantErr)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if r.calls != tc.wantCalls {
				t.Errorf("resolver calls = %d, want %d", r.calls, tc.wantCalls)
			}
		})
	}
}

func TestChecker_Check_allowlistSkipsListChecks(t *testing.T) {
	r := &fakeResolver{fn: okMX}
	c := newChecker(t, r, true)
	c.allow = domainSet{"gmial.com": {}}

	if err := c.Check(context.Background(), "a@gmial.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.calls != 1 {
		t.Errorf("resolver calls = %d, want 1", r.calls)
	}
}

func TestOSADistance(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal strings are 0", "gmail", "gmail", 0},
		{"insertion is 1", "gmail", "gmaill", 1},
		{"deletion is 1", "gmail", "gmal", 1},
		{"substitution is 1", "gmail", "gmaul", 1},
		{"adjacent swap is 1", "gmial", "gmail", 1},
		{"both empty is 0", "", "", 0},
		{"empty to string is its length", "", "abc", 3},
		{"string to empty is its length", "abc", "", 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := osaDistance(tc.a, tc.b); got != tc.want {
				t.Errorf("osaDistance(%q,%q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestDomainSet_matches(t *testing.T) {
	tests := []struct {
		name   string
		set    domainSet
		domain string
		want   bool
	}{
		{"matches exact domain", domainSet{"a.com": {}}, "a.com", true},
		{"matches parent domain", domainSet{"a.com": {}}, "x.y.a.com", true},
		{"does not match unrelated domain", domainSet{"a.com": {}}, "b.com", false},
		{"stops before bare TLD", domainSet{"com": {}}, "a.com", false},
		{"matches a bare entry only exactly", domainSet{"com": {}}, "com", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.set.matches(tc.domain); got != tc.want {
				t.Errorf("matches(%q) = %v, want %v", tc.domain, got, tc.want)
			}
		})
	}
}
