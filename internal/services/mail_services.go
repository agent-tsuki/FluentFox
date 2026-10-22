// Package services contains application-level services shared across domains.
// MailService renders the HTML emails in templates/ and hands them to the
// mailer.Mailer interface. The ctx methods wait for delivery so callers can
// react to a failure; the Async methods send in a goroutine and only log.
package services

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/mailer"
	"go.uber.org/zap"
)

const (
	verifyEmailSubject       = "Verify your FoxSensei account"
	passwordResetCodeSubject = "Your FoxSensei password reset code"
	passwordResetLinkSubject = "Reset your FoxSensei password" //nolint:gosec // an email subject, not a credential
	dailyReminderSubject     = "Don't break your streak! 🦊"
	loginCodeSubject         = "Your FoxSensei sign-in code"
	staffInviteSubject       = "You're invited to help run FoxSensei classes"

	verifyPath        = "/verify"
	invitePath        = "/invite/"
	hoursPerDay       = 24
	resetPasswordPath = "/reset-password"
	dashboardPath     = "/dashboard"

	ignoreIfNotYouNote = "Didn't ask for this? You can safely ignore this email."
)

// MailService holds the mailer (Resend in production, a stub in tests)
// and the base APP_URL used to build action links.
type MailService struct {
	mailer mailer.Mailer
	appURL string
	logger *zap.Logger
}

// appURL should be the frontend origin, e.g. "https://yourdomain.com".
func NewMailService(m mailer.Mailer, appURL string, log *zap.Logger) *MailService {
	return &MailService{mailer: m, appURL: appURL, logger: log}
}

// Waits for the result so callers can roll back the signup when delivery fails.
func (s *MailService) SendVerificationEmail(ctx context.Context, to, name, rawToken string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	html, err := s.verifyEmailHTML(name, rawToken)
	if err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	if err := s.mailer.SendHTML(ctx, to, verifyEmailSubject, html); err != nil {
		s.logger.Error("failed to send verification email", zap.Error(err))
		return fmt.Errorf("send verification email: %w", err)
	}
	s.logger.Info("verification email sent")
	return nil
}

// rawToken is the plaintext token. The link opens the frontend's /verify page,
// which then calls POST /auth/verify.
func (s *MailService) SendVerificationEmailAsync(to, name, rawToken string) {
	html, err := s.verifyEmailHTML(name, rawToken)
	if err != nil {
		s.logger.Error("async: failed to render verification email", zap.Error(err))
		return
	}
	s.sendAsync(to, verifyEmailSubject, html, verifyEmailTemplate)
}

func (s *MailService) SendPasswordResetEmailAsync(to, name, rawToken string) {
	html, err := s.passwordResetLinkHTML(name, rawToken)
	if err != nil {
		s.logger.Error("async: failed to render password reset email", zap.Error(err))
		return
	}
	s.sendAsync(to, passwordResetLinkSubject, html, passwordResetLinkTemplate)
}

func (s *MailService) SendPasswordResetCode(ctx context.Context, to, name, code string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	html, err := renderEmail(passwordResetCodeTemplate, passwordResetCodeData{
		emailLayout: newEmailLayout(s.appURL, passwordResetCodeSubject,
			"Your code to reset your FoxSensei password.", ignoreIfNotYouNote),
		Name:      displayName(name),
		Code:      code,
		ExpiresIn: humanDuration(common.PASSWORD_RESET_CODE_EXPIRES_AFTER),
	})
	if err != nil {
		return fmt.Errorf("send password reset code: %w", err)
	}
	if err := s.mailer.SendHTML(ctx, to, passwordResetCodeSubject, html); err != nil {
		s.logger.Error("failed to send password reset code", zap.Error(err))
		return fmt.Errorf("send password reset code: %w", err)
	}
	return nil
}

func (s *MailService) SendPasswordResetLink(ctx context.Context, to, name, rawToken string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	html, err := s.passwordResetLinkHTML(name, rawToken)
	if err != nil {
		return fmt.Errorf("send password reset link: %w", err)
	}
	if err := s.mailer.SendHTML(ctx, to, passwordResetLinkSubject, html); err != nil {
		s.logger.Error("failed to send password reset link", zap.Error(err))
		return fmt.Errorf("send password reset link: %w", err)
	}
	return nil
}

// where is a place such as "Tokyo, Japan", or "" when unknown.
func (s *MailService) SendLoginCode(ctx context.Context, to, name, code, where string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	data := loginCodeData{
		emailLayout: newEmailLayout(s.appURL, loginCodeSubject,
			"Enter this code to finish signing in to FoxSensei.", ignoreIfNotYouNote),
		Name:      displayName(name),
		Code:      code,
		ExpiresIn: humanDuration(common.LOGIN_CODE_EXPIRES_AFTER),
	}
	if where != "" {
		data.Where = " from " + where
	}
	html, err := renderEmail(loginCodeTemplate, data)
	if err != nil {
		return fmt.Errorf("send login code: %w", err)
	}
	if err := s.mailer.SendHTML(ctx, to, loginCodeSubject, html); err != nil {
		s.logger.Error("failed to send login code", zap.Error(err))
		return fmt.Errorf("send login code: %w", err)
	}
	return nil
}

// The token goes in the path because the frontend route is /invite/{token}
// and Google sign-in can return to that path.
func (s *MailService) SendStaffInvite(ctx context.Context, to, rawToken string) error {
	ctx, cancel := context.WithTimeout(ctx, common.EMAIL_SEND_TIMEOUT)
	defer cancel()

	html, err := renderEmail(staffInviteTemplate, staffInviteData{
		emailLayout: newEmailLayout(s.appURL, staffInviteSubject,
			"Accept your invite to edit FoxSensei class pages.",
			"Not expecting this? You can safely ignore this email."),
		ExpiresIn: plural(int(common.STAFF_INVITE_EXPIRES_AFTER/(hoursPerDay*time.Hour)), "day"),
		Button:    emailButton{URL: s.appURL + invitePath + url.PathEscape(rawToken), Label: "Accept invite"},
	})
	if err != nil {
		return fmt.Errorf("send staff invite: %w", err)
	}
	if err := s.mailer.SendHTML(ctx, to, staffInviteSubject, html); err != nil {
		s.logger.Error("failed to send staff invite", zap.Error(err))
		return fmt.Errorf("send staff invite: %w", err)
	}
	return nil
}

func (s *MailService) SendDailyReminderAsync(to, name string, streakDays int) {
	html, err := renderEmail(dailyReminderTemplate, dailyReminderData{
		emailLayout: newEmailLayout(s.appURL, dailyReminderSubject,
			"A few minutes today keeps your Japanese streak going.",
			"You're getting this because study reminders are on in your FoxSensei settings."),
		Name:       displayName(name),
		StreakDays: streakDays,
		Button:     emailButton{URL: s.appURL + dashboardPath, Label: "Continue learning"},
	})
	if err != nil {
		s.logger.Error("async: failed to render daily reminder", zap.Error(err))
		return
	}
	s.sendAsync(to, dailyReminderSubject, html, dailyReminderTemplate)
}

func (s *MailService) verifyEmailHTML(name, rawToken string) (string, error) {
	return renderEmail(verifyEmailTemplate, verifyEmailData{
		emailLayout: newEmailLayout(s.appURL, verifyEmailSubject,
			"Confirm your email to start learning Japanese with FoxSensei.",
			"Didn't create a FoxSensei account? You can safely ignore this email."),
		Name:      displayName(name),
		ExpiresIn: humanDuration(time.Duration(common.AUTH_TOKEN_EXPIRE_HOUR) * time.Hour),
		Button:    emailButton{URL: s.tokenURL(verifyPath, rawToken), Label: "Verify my email"},
	})
}

func (s *MailService) passwordResetLinkHTML(name, rawToken string) (string, error) {
	return renderEmail(passwordResetLinkTemplate, passwordResetLinkData{
		emailLayout: newEmailLayout(s.appURL, passwordResetLinkSubject,
			"Set a new password for your FoxSensei account.", ignoreIfNotYouNote),
		Name:      displayName(name),
		ExpiresIn: humanDuration(common.PASSWORD_RESET_LINK_EXPIRES_AFTER),
		Button:    emailButton{URL: s.tokenURL(resetPasswordPath, rawToken), Label: "Choose a new password"},
	})
}

func (s *MailService) tokenURL(path, rawToken string) string {
	return fmt.Sprintf("%s%s?token=%s", s.appURL, path, url.QueryEscape(rawToken))
}

// Async callers have no request to wait on, so the send gets its own bounded context.
func (s *MailService) sendAsync(to, subject, html, emailType string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), common.EMAIL_SEND_TIMEOUT)
		defer cancel()
		if err := s.mailer.SendHTML(ctx, to, subject, html); err != nil {
			s.logger.Error("async: failed to send email", zap.String("email_type", emailType), zap.Error(err))
		}
	}()
}
