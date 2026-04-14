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

// SendHTML sends a caller-rendered HTML email.
func (m *ResendMailer) SendHTML(ctx context.Context, to, subject, html string) error {
	params := &resend.SendEmailRequest{
		From:    m.fromAddr,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}
	_, err := m.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("mailer: send email to %s: %w", to, err)
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

func reminderEmailHTML(name string, streakDays int) string {
	return fmt.Sprintf(`<h1>Hey %s, keep your streak alive!</h1>
<p>You have a %d-day streak. Don't let it slip!</p>
<p><a href="https://fluentfox.app">Study now</a></p>`, name, streakDays)
}
