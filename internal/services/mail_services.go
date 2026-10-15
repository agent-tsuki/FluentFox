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
	"html"
	"net/url"
	"strings"

	"github.com/fluentfox/api/internal/common"
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

const verifyEmailSubject = "Verify your FoxSensei account"

// Waits for the result so callers can roll back the signup when delivery fails.
func (s *MailService) SendVerificationEmail(ctx context.Context, to, name, rawToken string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	verifyURL := fmt.Sprintf("%s/verify?token=%s", s.appURL, url.QueryEscape(rawToken))
	if err := s.mailer.SendHTML(ctx, to, verifyEmailSubject, renderVerifyEmail(name, verifyURL)); err != nil {
		s.logger.Error("failed to send verification email", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("send verification email: %w", err)
	}
	s.logger.Info("verification email sent", zap.String("to", to))
	return nil
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

func (s *MailService) SendPasswordResetCode(ctx context.Context, to, name, code string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	if err := s.mailer.SendHTML(ctx, to, "Your FoxSensei password reset code", passwordResetCodeEmail(name, code)); err != nil {
		s.logger.Error("failed to send password reset code", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("send password reset code: %w", err)
	}
	return nil
}

func (s *MailService) SendPasswordResetLink(ctx context.Context, to, name, rawToken string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, url.QueryEscape(rawToken))
	if err := s.mailer.SendHTML(ctx, to, "Reset your FoxSensei password", passwordResetLinkEmail(name, resetURL)); err != nil {
		s.logger.Error("failed to send password reset link", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("send password reset link: %w", err)
	}
	return nil
}

func passwordResetCodeEmail(name, code string) string {
	return fmt.Sprintf(`<h1>Password reset</h1><p>Hi %s,</p><p>Your password reset code is:</p><p style="font-size:24px;font-weight:bold;letter-spacing:4px">%s</p><p>This code expires in 10 minutes. If you did not request it, you can ignore this email.</p>`, html.EscapeString(displayName(name)), html.EscapeString(code))
}

func passwordResetLinkEmail(name, resetURL string) string {
	return fmt.Sprintf(`<h1>Reset your password</h1><p>Hi %s,</p><p><a href="%s">Reset your password</a></p><p>This link expires in 1 hour and can only be used once. If you did not request it, you can ignore this email.</p>`, html.EscapeString(displayName(name)), html.EscapeString(resetURL))
}

func displayName(name string) string {
	if name == "" {
		return "there"
	}
	return name
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
