package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/emailcheck"
	"github.com/fluentfox/api/pkg/exceptions"
)

type EmailChecker interface {
	Check(ctx context.Context, email string) error
}

type VerificationMailer interface {
	SendVerificationEmail(ctx context.Context, to, name, rawToken string) error
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// A DNS outage is logged and allowed, so it never blocks real signups.
func mapEmailCheckErr(err error, log *zap.Logger) error {
	var typo *emailcheck.TypoError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, emailcheck.ErrLookupUnavailable):
		log.Warn("email domain lookup unavailable, allowing address", zap.Error(err))
		return nil
	case errors.As(err, &typo):
		return exceptions.EmailDomainTypo(typo.Suggested)
	case errors.Is(err, emailcheck.ErrDisposable):
		return exceptions.ErrDisposableEmail
	case errors.Is(err, emailcheck.ErrNoMailServer):
		return exceptions.ErrInvalidEmailDomain
	case errors.Is(err, emailcheck.ErrMalformed):
		return exceptions.ErrValidation
	default:
		return fmt.Errorf("check email: %w", err)
	}
}
