package auth

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fluentfox/api/pkg/emailcheck"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestMapEmailCheckErr(t *testing.T) {
	tests := []struct {
		name     string
		in       error
		wantNil  bool
		wantCode string
		wantApp  *exceptions.AppError
	}{
		{name: "returns nil when check passed", in: nil, wantNil: true},
		{name: "returns nil when DNS lookup unavailable", in: fmt.Errorf("%w: boom", emailcheck.ErrLookupUnavailable), wantNil: true},
		{name: "maps typo to EMAIL_DOMAIN_TYPO with suggestion", in: &emailcheck.TypoError{Suggested: "a@gmail.com"}, wantCode: "EMAIL_DOMAIN_TYPO"},
		{name: "maps disposable to ErrDisposableEmail", in: emailcheck.ErrDisposable, wantApp: exceptions.ErrDisposableEmail},
		{name: "maps no mail server to ErrInvalidEmailDomain", in: emailcheck.ErrNoMailServer, wantApp: exceptions.ErrInvalidEmailDomain},
		{name: "maps malformed to ErrValidation", in: emailcheck.ErrMalformed, wantApp: exceptions.ErrValidation},
		{name: "maps wrapped malformed to ErrValidation", in: fmt.Errorf("%w: no dot", emailcheck.ErrMalformed), wantApp: exceptions.ErrValidation},
		{name: "wraps unknown error as non-AppError", in: errors.New("boom")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapEmailCheckErr(tc.in, nopLogger())
			if tc.wantNil {
				if got != nil {
					t.Fatalf("want nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("want error, got nil")
			}
			if tc.wantApp != nil && !errors.Is(got, tc.wantApp) {
				t.Errorf("got %v, want %v", got, tc.wantApp)
			}
			app, isApp := exceptions.As(got)
			if tc.wantCode != "" {
				if !isApp || app.Code != tc.wantCode {
					t.Errorf("got %v, want code %s", got, tc.wantCode)
				}
				if app.Message != "did you mean a@gmail.com?" {
					t.Errorf("message = %q", app.Message)
				}
			}
			if tc.wantApp == nil && tc.wantCode == "" {
				if isApp {
					t.Errorf("unknown error must not be an AppError, got %v", got)
				}
				if errors.Unwrap(got) == nil {
					t.Errorf("cause not wrapped: %v", got)
				}
			}
		})
	}
}
