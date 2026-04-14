// Package mailer — Gmail SMTP implementation of the Mailer interface.
// Uses stdlib net/smtp with STARTTLS on port 587.
// Auth is Gmail app password (not your account password).
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
)

// SMTPMailer implements Mailer using Gmail SMTP with an app password.
type SMTPMailer struct {
	host     string // e.g. smtp.gmail.com
	port     string // e.g. 587
	username string // full Gmail address
	password string // Gmail app password
	fromAddr string // displayed From address
}

// NewSMTPMailer constructs an SMTPMailer.
func NewSMTPMailer(host, port, username, password, fromAddr string) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		fromAddr: fromAddr,
	}
}

// SendHTML sends a caller-rendered HTML email over Gmail SMTP with STARTTLS.
func (m *SMTPMailer) SendHTML(_ context.Context, to, subject, html string) error {
	addr := net.JoinHostPort(m.host, m.port)

	// Build the raw MIME message.
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.fromAddr, to, subject, html,
	)

	// Dial plain TCP first — STARTTLS upgrades it.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp: dial %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS via STARTTLS.
	tlsCfg := &tls.Config{ServerName: m.host}
	if err := client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("smtp: starttls: %w", err)
	}

	// Authenticate with Gmail app password.
	auth := smtp.PlainAuth("", m.username, m.password, m.host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp: auth failed (check app password): %w", err)
	}

	if err := client.Mail(m.username); err != nil {
		return fmt.Errorf("smtp: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp: RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: DATA: %w", err)
	}
	if _, err := fmt.Fprint(w, msg); err != nil {
		return fmt.Errorf("smtp: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: close writer: %w", err)
	}

	return client.Quit()
}

// SendDailyReminder sends a plain study-reminder email.
func (m *SMTPMailer) SendDailyReminder(_ context.Context, to, name string, streakDays int) error {
	html := fmt.Sprintf(
		`<h1>Hey %s, keep your streak alive!</h1><p>You have a %d-day streak. Don't let it slip!</p><p><a href="https://fluentfox.app">Study now</a></p>`,
		name, streakDays,
	)
	return m.SendHTML(context.Background(), to, "Don't break your streak!", html)
}
