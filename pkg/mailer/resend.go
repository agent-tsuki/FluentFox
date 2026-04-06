// Package mailer — Resend implementation of the Mailer interface.
// This file is the only place in the codebase that imports resend-go.
// All other packages use the Mailer interface.
package mailer

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// ResendMailer implements Mailer using the Resend transactional email API.
type ResendMailer struct {
	client   *resend.Client
	fromAddr string
}

// NewResendMailer constructs a ResendMailer.
// apiKey and fromAddress come from Config — never from env directly.
func NewResendMailer(apiKey, fromAddress string) *ResendMailer {
	return &ResendMailer{
		client:   resend.NewClient(apiKey),
		fromAddr: fromAddress,
	}
}

// SendVerificationEmail sends an HTML email with the account verification link.
func (m *ResendMailer) SendVerificationEmail(ctx context.Context, to, name, verifyURL string) error {
	params := &resend.SendEmailRequest{
		From:    m.fromAddr,
		To:      []string{to},
		Subject: "Verify your FluentFox account",
		Html:    verifyEmailHTML(name, verifyURL),
	}

	_, err := m.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("mailer: send verification email to %s: %w", to, err)
	}
	return nil
}

// SendPasswordReset sends an HTML email with the password reset link.
func (m *ResendMailer) SendPasswordReset(ctx context.Context, to, name, resetURL string) error {
	params := &resend.SendEmailRequest{
		From:    m.fromAddr,
		To:      []string{to},
		Subject: "Reset your FluentFox password",
		Html:    resetEmailHTML(name, resetURL),
	}

	_, err := m.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("mailer: send password reset email to %s: %w", to, err)
	}
	return nil
}

// SendDailyReminder sends a study reminder email.
func (m *ResendMailer) SendDailyReminder(ctx context.Context, to, name string, streakDays int) error {
	params := &resend.SendEmailRequest{
		From:    m.fromAddr,
		To:      []string{to},
		Subject: "Don't break your streak! 🦊",
		Html:    reminderEmailHTML(name, streakDays),
	}

	_, err := m.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("mailer: send daily reminder to %s: %w", to, err)
	}
	return nil
}

func verifyEmailHTML(name, verifyURL string) string {
	return fmt.Sprintf(`<h1>Welcome to FluentFox, %s!</h1>
<p>Click the link below to verify your email address:</p>
<p><a href="%s">Verify my account</a></p>
<p>This link expires in 24 hours.</p>`, name, verifyURL)
}

func resetEmailHTML(name, resetURL string) string {
	return fmt.Sprintf(`<h1>Password reset for %s</h1>
<p>Click the link below to reset your password:</p>
<p><a href="%s">Reset my password</a></p>
<p>This link expires in 1 hour. If you did not request this, ignore this email.</p>`, name, resetURL)
}

func reminderEmailHTML(name string, streakDays int) string {
	return fmt.Sprintf(`<h1>Hey %s, keep your streak alive!</h1>
<p>You have a %d-day streak. Don't let it slip!</p>
<p><a href="https://fluentfox.app">Study now</a></p>`, name, streakDays)
}
