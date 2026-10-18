package services

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"sync"
	"time"

	"github.com/fluentfox/api/internal/common"
)

//go:embed templates/*.html
var templateFS embed.FS

const (
	verifyEmailTemplate       = "verify_email.html"
	passwordResetCodeTemplate = "password_reset_code.html"
	passwordResetLinkTemplate = "password_reset_link.html"
	dailyReminderTemplate     = "daily_reminder.html"
	serverErrorAlertTemplate  = "server_error_alert.html"
	loginCodeTemplate         = "login_code.html"
)

// html/template escapes every value, so a name like "<a href=…>" shows as
// text instead of becoming a link in someone's inbox. Parsed on first use so a
// bad template fails the send instead of panicking at startup.
var loadEmailTemplates = sync.OnceValues(func() (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/*.html")
})

// Fields read by layout.html.
type emailLayout struct {
	Title        string
	Preheader    string
	FooterNote   string
	AppURL       string
	LogoURL      string
	SupportEmail string
}

type emailButton struct {
	URL   string
	Label string
}

type verifyEmailData struct {
	emailLayout
	Name      string
	ExpiresIn string
	Button    emailButton
}

type passwordResetCodeData struct {
	emailLayout
	Name      string
	Code      string
	ExpiresIn string
}

type passwordResetLinkData struct {
	emailLayout
	Name      string
	ExpiresIn string
	Button    emailButton
}

type loginCodeData struct {
	emailLayout
	Name      string
	Code      string
	ExpiresIn string
	// " from Tokyo, Japan" or "" when the location is unknown.
	Where string
}

type dailyReminderData struct {
	emailLayout
	Name       string
	StreakDays int
	Button     emailButton
}

func newEmailLayout(appURL, title, preheader, footerNote string) emailLayout {
	return emailLayout{
		Title:        title,
		Preheader:    preheader,
		FooterNote:   footerNote,
		AppURL:       appURL,
		LogoURL:      common.EMAIL_LOGO_URL,
		SupportEmail: common.EMAIL_SUPPORT_ADDRESS,
	}
}

func renderEmail(name string, data any) (string, error) {
	tmpl, err := loadEmailTemplates()
	if err != nil {
		return "", fmt.Errorf("parse email templates: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return buf.String(), nil
}

// Turns 6h into "6 hours" and 10m into "10 minutes" for email copy.
func humanDuration(d time.Duration) string {
	if d >= time.Hour && d%time.Hour == 0 {
		return plural(int(d/time.Hour), "hour")
	}
	return plural(int(d/time.Minute), "minute")
}

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

func displayName(name string) string {
	if name == "" {
		return "there"
	}
	return name
}
