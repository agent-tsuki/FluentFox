package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

const passwordResetTestSecret = "0123456789abcdef0123456789abcdef"

func newPasswordResetService(repo UserRepo, mailer PasswordResetMailer) *PasswordResetService {
	service := NewPasswordResetService(repo, mailer, passwordResetTestSecret, nopLogger())
	service.argon = Argon2Config{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	return service
}

func activePasswordResetUser(id uuid.UUID) users.User {
	return users.User{ID: id, Email: "haruki@example.com", IsActive: true}
}

func TestPasswordResetService_RequestCode(t *testing.T) {
	uid := uuid.New()
	tests := []struct {
		name          string
		repo          *fakeUserRepo
		wantMailCount int
		wantStored    bool
	}{
		{
			name: "sends an uppercase code and stores only its keyed hash",
			repo: &fakeUserRepo{
				byEmail: map[string]users.User{"haruki@example.com": activePasswordResetUser(uid)},
				locked:  activePasswordResetUser(uid),
				profile: users.UserProfile{FirstName: "Haruki"},
			},
			wantMailCount: 1,
			wantStored:    true,
		},
		{
			name:          "returns the same success result for an unknown email",
			repo:          &fakeUserRepo{emailErr: gorm.ErrRecordNotFound},
			wantMailCount: 0,
		},
		{
			name: "suppresses a request during the resend cooldown",
			repo: &fakeUserRepo{
				byEmail:       map[string]users.User{"haruki@example.com": activePasswordResetUser(uid)},
				locked:        activePasswordResetUser(uid),
				profile:       users.UserProfile{FirstName: "Haruki"},
				passwordReset: users.PasswordReset{CodeLastSentAt: time.Now().UTC()},
			},
			wantMailCount: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &fakeMailer{}
			service := newPasswordResetService(tc.repo, mailer)

			if err := service.RequestCode(context.Background(), ForgotPasswordRequest{Email: "  HARUKI@example.com "}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mailer.resetCodes) != tc.wantMailCount {
				t.Fatalf("sent codes = %d, want %d", len(mailer.resetCodes), tc.wantMailCount)
			}
			if !tc.wantStored {
				return
			}

			code := mailer.resetCodes[0].token
			if len(code) != common.PASSWORD_RESET_CODE_LENGTH || strings.Trim(code, passwordResetCodeCharset) != "" {
				t.Fatalf("code %q is not uppercase alphanumeric", code)
			}
			if tc.repo.storedCodeHash == code || tc.repo.storedCodeHash != service.hashCode(code) {
				t.Fatal("stored code must be the keyed hash, never the raw code")
			}
			if tc.repo.storedSentCount != 1 {
				t.Fatalf("send count = %d, want 1", tc.repo.storedSentCount)
			}
		})
	}
}

func TestPasswordResetService_VerifyCode(t *testing.T) {
	uid := uuid.New()
	serviceCode := "K7M9Q2XR"
	tests := []struct {
		name        string
		code        string
		reset       users.PasswordReset
		wantErr     error
		wantLinks   int
		wantAttempt int
	}{
		{
			name:      "sends a one-time reset link after a valid code",
			code:      serviceCode,
			reset:     users.PasswordReset{CodeHash: newPasswordResetService(&fakeUserRepo{}, &fakeMailer{}).hashCode(serviceCode), CodeExpiresAt: time.Now().UTC().Add(time.Minute)},
			wantLinks: 1,
		},
		{
			name:        "records a failed code attempt without sending a link",
			code:        "AAAAAAAA",
			reset:       users.PasswordReset{CodeHash: newPasswordResetService(&fakeUserRepo{}, &fakeMailer{}).hashCode(serviceCode), CodeExpiresAt: time.Now().UTC().Add(time.Minute)},
			wantErr:     exceptions.ErrPasswordResetCodeInvalid,
			wantAttempt: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &fakeMailer{}
			repo := &fakeUserRepo{
				byEmail:       map[string]users.User{"haruki@example.com": activePasswordResetUser(uid)},
				locked:        activePasswordResetUser(uid),
				profile:       users.UserProfile{FirstName: "Haruki"},
				passwordReset: tc.reset,
			}
			service := newPasswordResetService(repo, mailer)

			err := service.VerifyCode(context.Background(), VerifyPasswordResetCodeRequest{Email: "haruki@example.com", Code: tc.code})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mailer.resetLinks) != tc.wantLinks {
				t.Fatalf("sent links = %d, want %d", len(mailer.resetLinks), tc.wantLinks)
			}
			if repo.storedCodeAttempts != tc.wantAttempt {
				t.Fatalf("attempts = %d, want %d", repo.storedCodeAttempts, tc.wantAttempt)
			}
			if tc.wantLinks == 1 && (repo.storedLinkHash == mailer.resetLinks[0].token || repo.storedLinkExpiresAt.IsZero()) {
				t.Fatal("stored link must be hashed with an expiry")
			}
		})
	}
}

func TestPasswordResetService_ResetPassword(t *testing.T) {
	uid := uuid.New()
	rawToken := "one-time-reset-token"
	repo := &fakeUserRepo{
		findResetUserID: uid,
		consumed:        true,
		locked:          activePasswordResetUser(uid),
		passwordReset: users.PasswordReset{
			ResetTokenHash:      ptrString(hashVerificationToken(rawToken)),
			ResetTokenExpiresAt: ptrTime(time.Now().UTC().Add(time.Minute)),
		},
	}
	service := newPasswordResetService(repo, &fakeMailer{})

	if err := service.ResetPassword(context.Background(), ResetPasswordRequest{Token: rawToken, NewPassword: "new-correct-password"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.consumedPasswordHash == "new-correct-password" || repo.consumedPasswordHash == "" {
		t.Fatal("new password must be hashed before it is stored")
	}
}

func ptrString(value string) *string { return &value }
