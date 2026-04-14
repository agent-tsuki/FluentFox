// Package services contains application-level services shared across domains.
// MailService is a fire-and-forget wrapper around the mailer.Mailer interface.
// Every Send* method starts a goroutine so the calling handler can return 200
// immediately — mail delivery failures are logged, and the user can retry via
// the resend-verification endpoint.
package services

import (
	"context"
	_ "embed"
	"fmt"
	"net/url"
	"strings"

	"github.com/fluentfox/api/pkg/mailer"
	"go.uber.org/zap"
)

//go:embed templates/verify_email.html
var verifyEmailTemplate string

// MailService dispatches emails asynchronously.
// It holds a reference to the mailer interface (Resend in production, a no-op
// stub in tests) and the base APP_URL used to build action links.
type MailService struct {
	mailer mailer.Mailer
	appURL string
	logger *zap.Logger
}

// NewMailService constructs a MailService.
// appURL should be the frontend origin, e.g. "https://yourdomain.com".
func NewMailService(m mailer.Mailer, appURL string, log *zap.Logger) *MailService {
	return &MailService{mailer: m, appURL: appURL, logger: log}
}

// renderVerifyEmail replaces {{name}} and {{verify_url}} in the embedded template.
func renderVerifyEmail(name, verifyURL string) string {
	return strings.NewReplacer(
		"{{name}}", name,
		"{{verify_url}}", verifyURL,
	).Replace(verifyEmailTemplate)
}

// SendVerificationEmailAsync fires the verification email in a background
// goroutine and returns immediately.
//
// rawToken is the plaintext token — it is URL-encoded and appended as
// APP_URL + "/verify?token=..." so the user's browser hits the frontend,
// which then calls POST /auth/verify.
func (s *MailService) SendVerificationEmailAsync(to, name, rawToken string) {
	verifyURL := fmt.Sprintf("%s/verify?token=%s", s.appURL, url.QueryEscape(rawToken))
	html := renderVerifyEmail(name, verifyURL)
	go func() {
		if err := s.mailer.SendHTML(context.Background(), to, "Verify your FoxSensei account", html); err != nil {
			s.logger.Error("async: failed to send verification email",
				zap.String("to", to),
				zap.Error(err),
			)
		} else {
			s.logger.Info("async: verification email sent", zap.String("to", to))
		}
	}()
}

// SendPasswordResetEmailAsync fires the password-reset email in a background goroutine.
// The link points to APP_URL + "/reset-password?token=..."
func (s *MailService) SendPasswordResetEmailAsync(to, name, rawToken string) {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, url.QueryEscape(rawToken))
	html := fmt.Sprintf(`<h1>Password reset for %s</h1>
<p>Click the link below to reset your password:</p>
<p><a href="%s">Reset my password</a></p>
<p>This link expires in 1 hour. If you did not request this, ignore this email.</p>`, name, resetURL)
	go func() {
		if err := s.mailer.SendHTML(context.Background(), to, "Reset your FoxSensei password", html); err != nil {
			s.logger.Error("async: failed to send password reset email",
				zap.String("to", to),
				zap.Error(err),
			)
		} else {
			s.logger.Info("async: password reset email sent", zap.String("to", to))
		}
	}()
}

// SendDailyReminderAsync fires a study-reminder email in a background goroutine.
func (s *MailService) SendDailyReminderAsync(to, name string, streakDays int) {
	go func() {
		if err := s.mailer.SendDailyReminder(context.Background(), to, name, streakDays); err != nil {
			s.logger.Error("async: failed to send daily reminder",
				zap.String("to", to),
				zap.Error(err),
			)
		}
	}()
}
