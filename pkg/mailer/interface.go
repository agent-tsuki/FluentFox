// Package mailer defines the Mailer interface and its concrete implementations.
// Services depend only on this interface — never on Resend SDK types.
package mailer

import "context"

// Mailer is the contract every email sender must implement.
// It is a pure transport — callers are responsible for rendering HTML before calling.
type Mailer interface {
	// SendHTML sends an arbitrary HTML email. subject and html are caller-rendered.
	SendHTML(ctx context.Context, to, subject, html string) error

	// SendDailyReminder sends a study reminder to inactive users.
	SendDailyReminder(ctx context.Context, to, name string, streakDays int) error
}
