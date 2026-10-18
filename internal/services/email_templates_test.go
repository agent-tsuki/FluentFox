package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/alert"
)

const testAppURL = "https://foxsensei.com"

type sentEmail struct {
	to, subject, html string
}

type fakeMailer struct {
	sent []sentEmail
}

func (f *fakeMailer) SendHTML(_ context.Context, to, subject, html string) error {
	f.sent = append(f.sent, sentEmail{to: to, subject: subject, html: html})
	return nil
}

func lastEmail(t *testing.T, m *fakeMailer) sentEmail {
	t.Helper()
	if len(m.sent) == 0 {
		t.Fatal("no email was sent")
	}
	return m.sent[len(m.sent)-1]
}

func TestMailService_RendersEveryEmailWithLayout(t *testing.T) {
	tests := []struct {
		name         string
		send         func(s *MailService) error
		wantSubject  string
		wantContains []string
	}{
		{
			name: "verification email links to the verify page",
			send: func(s *MailService) error {
				return s.SendVerificationEmail(context.Background(), "a@b.com", "Aiko", "tok en")
			},
			wantSubject: verifyEmailSubject,
			wantContains: []string{
				"Hi Aiko, welcome to FoxSensei",
				testAppURL + "/verify?token=tok&#43;en",
				"6 hours",
			},
		},
		{
			name: "password reset code shows the code and expiry",
			send: func(s *MailService) error {
				return s.SendPasswordResetCode(context.Background(), "a@b.com", "", "AB12CD34")
			},
			wantSubject:  passwordResetCodeSubject,
			wantContains: []string{"Hi there,", "AB12CD34", "10 minutes"},
		},
		{
			name: "login code shows the code and where the sign-in came from",
			send: func(s *MailService) error {
				return s.SendLoginCode(context.Background(), "a@b.com", "Aiko", "123456", "Tokyo, Japan")
			},
			wantSubject:  loginCodeSubject,
			wantContains: []string{"123456", "through a VPN or proxy from Tokyo, Japan", "10 minutes"},
		},
		{
			name: "password reset link links to the reset page",
			send: func(s *MailService) error {
				return s.SendPasswordResetLink(context.Background(), "a@b.com", "Aiko", "abc")
			},
			wantSubject:  passwordResetLinkSubject,
			wantContains: []string{testAppURL + "/reset-password?token=abc", "1 hour"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &fakeMailer{}
			if err := tt.send(NewMailService(m, testAppURL, zap.NewNop())); err != nil {
				t.Fatalf("send: %v", err)
			}
			got := lastEmail(t, m)
			if got.subject != tt.wantSubject {
				t.Errorf("subject = %q, want %q", got.subject, tt.wantSubject)
			}
			for _, want := range append(tt.wantContains, common.EMAIL_LOGO_URL, common.EMAIL_SUPPORT_ADDRESS) {
				if !strings.Contains(got.html, want) {
					t.Errorf("html does not contain %q", want)
				}
			}
		})
	}
}

func TestMailService_EscapesUserNames(t *testing.T) {
	m := &fakeMailer{}
	s := NewMailService(m, testAppURL, zap.NewNop())

	if err := s.SendPasswordResetLink(context.Background(), "a@b.com", `<a href="https://evil.example">x</a>`, "abc"); err != nil {
		t.Fatalf("send: %v", err)
	}

	if html := lastEmail(t, m).html; strings.Contains(html, `<a href="https://evil.example">`) {
		t.Error("user name was inserted as HTML")
	}
}

func TestServerErrorAlertMailer_SendsErrorDetails(t *testing.T) {
	m := &fakeMailer{}
	a := NewServerErrorAlertMailer(m, "auth@foxsensei.com", testAppURL, "production")

	err := a.NotifyServerError(context.Background(), &alert.ServerError{
		Time:       time.Date(2026, 10, 18, 15, 47, 44, 0, time.UTC),
		RequestID:  "3bb68b30",
		Error:      `login: load user: relation "users" does not exist`,
		Caller:     "humautil/errors.go:69",
		Stack:      "github.com/fluentfox/api/pkg/humautil.MapErr",
		Suppressed: 2,
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}

	got := lastEmail(t, m)
	if got.to != "auth@foxsensei.com" {
		t.Errorf("to = %q, want auth@foxsensei.com", got.to)
	}
	if !strings.HasPrefix(got.subject, "[FoxSensei production] 500: login: load user") {
		t.Errorf("subject = %q", got.subject)
	}
	for _, want := range []string{"3bb68b30", "relation &#34;users&#34; does not exist", "2026-10-18 15:47:44 UTC", "humautil.MapErr", "2 more error(s)"} {
		if !strings.Contains(got.html, want) {
			t.Errorf("html does not contain %q", want)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "whole hours", in: 6 * time.Hour, want: "6 hours"},
		{name: "one hour is singular", in: time.Hour, want: "1 hour"},
		{name: "minutes", in: 10 * time.Minute, want: "10 minutes"},
		{name: "ninety minutes stays in minutes", in: 90 * time.Minute, want: "90 minutes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humanDuration(tt.in); got != tt.want {
				t.Errorf("humanDuration(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
