// Package mailer defines the Mailer interface and its concrete implementations.
// Services depend only on this interface — never on Resend SDK types.
package mailer

import "context"

// Mailer is the contract every email sender must implement.
// All method arguments are plain Go types — no SDK types cross the boundary.
type Mailer interface {
	// SendVerificationEmail dispatches an account verification email.
	SendVerificationEmail(ctx context.Context, to, name, verifyURL string) error

	// SendPasswordReset dispatches a password reset link.
	SendPasswordReset(ctx context.Context, to, name, resetURL string) error

	// SendDailyReminder sends a study reminder to inactive users.
	SendDailyReminder(ctx context.Context, to, name string, streakDays int) error
}
