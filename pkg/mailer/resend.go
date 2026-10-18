// Package mailer — Resend implementation of the Mailer interface.
// This file is the only place in the codebase that imports resend-go.
// All other packages use the Mailer interface.
package mailer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/resend/resend-go/v2"
)

// ResendMailer implements Mailer using the Resend transactional email API.
type ResendMailer struct {
	client   *resend.Client
	fromAddr string
}

// NewResendMailer constructs a ResendMailer.
// apiKey and fromAddress come from Config — never from env directly.
// resend.NewClient uses http.DefaultClient, which never times out, so a hung
// API call could outlive the request; the timeout caps it even without a ctx deadline.
func NewResendMailer(apiKey, fromAddress string, timeout time.Duration) *ResendMailer {
	return &ResendMailer{
		client:   resend.NewCustomClient(&http.Client{Timeout: timeout}, apiKey),
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
	_, err := m.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("mailer: send email: %w", err)
	}
	return nil
}
