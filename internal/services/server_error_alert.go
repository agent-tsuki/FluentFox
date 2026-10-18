package services

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/fluentfox/api/pkg/alert"
	"github.com/fluentfox/api/pkg/mailer"
)

const (
	serverErrorAlertSubjectFormat = "[FoxSensei %s] 500: %s"
	// Keeps the subject readable in an inbox list; the full error is in the body.
	serverErrorSubjectMaxRunes = 80
	serverErrorTimeLayout      = "2006-01-02 15:04:05 MST"
)

type serverErrorAlertData struct {
	emailLayout
	alert.ServerError
	Env     string
	TimeUTC string
}

// Implements alert.Notifier by emailing one address.
type ServerErrorAlertMailer struct {
	mailer mailer.Mailer
	to     string
	appURL string
	env    string
}

func NewServerErrorAlertMailer(m mailer.Mailer, to, appURL, env string) *ServerErrorAlertMailer {
	return &ServerErrorAlertMailer{mailer: m, to: to, appURL: appURL, env: env}
}

func (a *ServerErrorAlertMailer) NotifyServerError(ctx context.Context, e *alert.ServerError) error {
	subject := fmt.Sprintf(serverErrorAlertSubjectFormat, a.env, truncateRunes(firstLine(e.Error), serverErrorSubjectMaxRunes))
	html, err := renderEmail(serverErrorAlertTemplate, serverErrorAlertData{
		emailLayout: newEmailLayout(a.appURL, subject, e.Error,
			fmt.Sprintf("Sent because ALERT_EMAIL_TO is set on the %s API.", a.env)),
		ServerError: *e,
		Env:         a.env,
		TimeUTC:     e.Time.UTC().Format(serverErrorTimeLayout),
	})
	if err != nil {
		return fmt.Errorf("server error alert: %w", err)
	}
	if err := a.mailer.SendHTML(ctx, a.to, subject, html); err != nil {
		return fmt.Errorf("server error alert: %w", err)
	}
	return nil
}

// A \r or \n from an upstream error must not reach the subject header.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, line)
}

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}

var _ alert.Notifier = (*ServerErrorAlertMailer)(nil)
