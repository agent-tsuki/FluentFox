package auth

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

func ptrTime(t time.Time) *time.Time { return &t }

func TestVerificationSender_Send(t *testing.T) {
	now := time.Now().UTC()
	boom := errors.New("smtp down")

	tests := []struct {
		name          string
		repo          *fakeUserRepo
		mailErr       error
		wantErr       *exceptions.AppError
		wantRetryMin  time.Duration
		wantRetryMax  time.Duration
		wantCalls     []string
		wantMailCount int
		wantCause     error
	}{
		{
			name:          "sends when there are no previous sends",
			repo:          &fakeUserRepo{},
			wantCalls:     []string{"ExpireActiveVerifications", "CreateUserVerification", "mail"},
			wantMailCount: 1,
		},
		{
			name: "sends when latest send was 2 minutes ago",
			repo: &fakeUserRepo{stats: users.VerificationSendStats{
				Count: 1, Oldest: ptrTime(now.Add(-2 * time.Minute)), Latest: ptrTime(now.Add(-2 * time.Minute)),
			}},
			wantCalls:     []string{"ExpireActiveVerifications", "CreateUserVerification", "mail"},
			wantMailCount: 1,
		},
		{
			name: "returns ErrVerificationCooldown when latest send was 10s ago",
			repo: &fakeUserRepo{stats: users.VerificationSendStats{
				Count: 1, Oldest: ptrTime(now.Add(-10 * time.Second)), Latest: ptrTime(now.Add(-10 * time.Second)),
			}},
			wantErr:      exceptions.ErrVerificationCooldown,
			wantRetryMin: 48 * time.Second,
			wantRetryMax: 51 * time.Second,
		},
		{
			name: "returns ErrVerificationDailyLimit when 5 sends in window",
			repo: &fakeUserRepo{stats: users.VerificationSendStats{
				Count: 5, Oldest: ptrTime(now.Add(-time.Hour)), Latest: ptrTime(now.Add(-30 * time.Minute)),
			}},
			wantErr:      exceptions.ErrVerificationDailyLimit,
			wantRetryMin: 22 * time.Hour,
			wantRetryMax: 23 * time.Hour,
		},
		{
			name:    "returns EMAIL_SEND_FAILED when mail fails, keeping cause",
			repo:    &fakeUserRepo{},
			mailErr: boom,
			wantErr: exceptions.ErrEmailSendFailed,
			// token rotation already happened inside the caller's transaction
			wantCalls:     []string{"ExpireActiveVerifications", "CreateUserVerification", "mail"},
			wantMailCount: 1,
			wantCause:     boom,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &fakeMailer{err: tc.mailErr}
			mailer.onSend = func() { tc.repo.record("mail") }
			s := newVerificationSender(tc.repo, mailer)
			uid := uuid.New()

			err := s.send(context.Background(), nil, uid, "a@b.com", "Ann")

			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != nil {
				app, ok := exceptions.As(err)
				if !ok || app.Code != tc.wantErr.Code || app.Status != tc.wantErr.Status {
					t.Fatalf("err = %v, want code %s", err, tc.wantErr.Code)
				}
				if tc.wantRetryMax > 0 && (app.RetryAfter < tc.wantRetryMin || app.RetryAfter > tc.wantRetryMax) {
					t.Errorf("RetryAfter = %v, want within [%v, %v]", app.RetryAfter, tc.wantRetryMin, tc.wantRetryMax)
				}
				if tc.wantCause != nil && !errors.Is(err, tc.wantCause) {
					t.Errorf("cause lost: %v", err)
				}
			}
			if !reflect.DeepEqual(tc.repo.calls, tc.wantCalls) {
				t.Errorf("calls = %v, want %v", tc.repo.calls, tc.wantCalls)
			}
			if len(mailer.sent) != tc.wantMailCount {
				t.Fatalf("mails = %d, want %d", len(mailer.sent), tc.wantMailCount)
			}
			if tc.wantMailCount > 0 && (mailer.sent[0].to != "a@b.com" || mailer.sent[0].name != "Ann" || mailer.sent[0].token == "") {
				t.Errorf("unexpected mail %+v", mailer.sent[0])
			}
		})
	}
}

func TestVerificationSender_Send_repoErrors(t *testing.T) {
	boom := errors.New("db down")
	tests := []struct {
		name string
		repo *fakeUserRepo
	}{
		{"wraps stats error", &fakeUserRepo{statsErr: boom}},
		{"wraps expire error", &fakeUserRepo{expireErr: boom}},
		{"wraps create verification error", &fakeUserRepo{createVerErr: boom}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &fakeMailer{}
			s := newVerificationSender(tc.repo, mailer)

			err := s.send(context.Background(), nil, uuid.New(), "a@b.com", "Ann")

			if !errors.Is(err, boom) {
				t.Errorf("err = %v, want wrapped %v", err, boom)
			}
			if len(mailer.sent) != 0 {
				t.Error("mail must not be sent when the repo fails")
			}
		})
	}
}
