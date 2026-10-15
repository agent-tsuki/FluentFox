package auth

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/emailcheck"
	"github.com/fluentfox/api/pkg/exceptions"
)

func assertAppCode(t *testing.T, err error, wantCode string) {
	t.Helper()
	app, ok := exceptions.As(err)
	if !ok {
		t.Fatalf("want AppError %s, got %v", wantCode, err)
	}
	if app.Code != wantCode {
		t.Fatalf("code = %s, want %s (%v)", app.Code, wantCode, err)
	}
}

func TestAuthService_registerUser(t *testing.T) {
	username := "bob"
	holderID := uuid.New()
	boom := errors.New("smtp down")

	tests := []struct {
		name          string
		repo          *fakeUserRepo
		checkerErr    error
		mailErr       error
		wantCode      string
		wantErr       *exceptions.AppError
		wantCreated   bool
		wantDeleted   bool
		wantMailCount int
		wantCause     error
	}{
		{
			name:          "creates the account and sends mail on the happy path",
			repo:          &fakeUserRepo{},
			wantCreated:   true,
			wantMailCount: 1,
		},
		{
			name:       "returns EMAIL_DOMAIN_TYPO when checker finds a typo",
			repo:       &fakeUserRepo{},
			checkerErr: &emailcheck.TypoError{Suggested: "bob@gmail.com"},
			wantCode:   "EMAIL_DOMAIN_TYPO",
		},
		{
			name: "returns ErrEmailAlreadyInUse when a verified user holds the email",
			repo: &fakeUserRepo{byEmail: map[string]users.User{
				"bob@gmail.com": {ID: holderID, IsEmailVerified: true},
			}},
			wantErr: exceptions.ErrEmailAlreadyInUse,
		},
		{
			name: "releases an unverified stale email holder and signs up",
			repo: &fakeUserRepo{
				byEmail: map[string]users.User{"bob@gmail.com": {ID: holderID}},
				locked:  users.User{ID: holderID},
				deleted: true,
			},
			wantDeleted:   true,
			wantCreated:   true,
			wantMailCount: 1,
		},
		{
			name: "returns 409 when unverified holder still has an active token",
			repo: &fakeUserRepo{
				byEmail:   map[string]users.User{"bob@gmail.com": {ID: holderID}},
				locked:    users.User{ID: holderID},
				hasActive: true,
			},
			wantErr: exceptions.ErrEmailAlreadyInUse,
		},
		{
			name: "returns ErrUsernameAlreadyInUse when a verified user holds the username",
			repo: &fakeUserRepo{byUsername: map[string]users.User{
				"bob": {ID: holderID, IsEmailVerified: true},
			}},
			wantErr: exceptions.ErrUsernameAlreadyInUse,
		},
		{
			name: "releases a stale username holder and signs up",
			repo: &fakeUserRepo{
				byUsername: map[string]users.User{"bob": {ID: holderID}},
				locked:     users.User{ID: holderID},
				deleted:    true,
			},
			wantDeleted:   true,
			wantCreated:   true,
			wantMailCount: 1,
		},
		{
			name:          "returns EMAIL_SEND_FAILED when mail fails",
			repo:          &fakeUserRepo{},
			mailErr:       boom,
			wantErr:       exceptions.ErrEmailSendFailed,
			wantCreated:   true,
			wantMailCount: 1,
			wantCause:     boom,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checker := &fakeEmailChecker{err: tc.checkerErr}
			mailer := &fakeMailer{err: tc.mailErr}
			releaser := NewStaleAccountReleaser(tc.repo, nopLogger())
			svc := NewAuthService(tc.repo, mailer, checker, releaser, nopLogger())
			req := RegisterRequest{
				Email: "  Bob@GMAIL.com ", UserName: &username, FirstName: "Bob",
				NativeLang: "en", Password: "password123",
			}

			err := svc.registerUser(context.Background(), req)

			if !reflect.DeepEqual(checker.checked, []string{"bob@gmail.com"}) {
				t.Errorf("checker saw %v, want normalized email", checker.checked)
			}
			switch {
			case tc.wantCode != "":
				assertAppCode(t, err, tc.wantCode)
			case tc.wantErr != nil:
				assertAppCode(t, err, tc.wantErr.Code)
				if !errors.Is(err, tc.wantErr) && tc.wantCause == nil {
					t.Errorf("err = %v, want %v", err, tc.wantErr)
				}
			case err != nil:
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantCause != nil && !errors.Is(err, tc.wantCause) {
				t.Errorf("cause lost: %v", err)
			}

			created := slices.Contains(tc.repo.calls, "CreateUser")
			if created != tc.wantCreated {
				t.Errorf("CreateUser called = %v, want %v", created, tc.wantCreated)
			}
			if created && tc.repo.createdEmail != "bob@gmail.com" {
				t.Errorf("created email = %q, want normalized", tc.repo.createdEmail)
			}
			if deleted := slices.Contains(tc.repo.calls, "DeleteUnverifiedUser"); deleted != tc.wantDeleted {
				t.Errorf("DeleteUnverifiedUser called = %v, want %v", deleted, tc.wantDeleted)
			}
			if len(mailer.sent) != tc.wantMailCount {
				t.Fatalf("mails = %d, want %d", len(mailer.sent), tc.wantMailCount)
			}
			if tc.wantMailCount > 0 && mailer.sent[0].to != "bob@gmail.com" {
				t.Errorf("mail sent to %q", mailer.sent[0].to)
			}
		})
	}
}

func TestResendVerificationService_ResendForUser(t *testing.T) {
	active := users.User{ID: uuid.New(), Email: "old@x.com", IsActive: true}

	tests := []struct {
		name     string
		repo     *fakeUserRepo
		wantErr  *exceptions.AppError
		wantMail string
	}{
		{
			name:    "returns ErrUserNotFound when user does not exist",
			repo:    &fakeUserRepo{byIDErr: gorm.ErrRecordNotFound},
			wantErr: exceptions.ErrUserNotFound,
		},
		{
			name:    "returns ErrEmailAlreadyVerified when user is verified",
			repo:    &fakeUserRepo{byID: users.User{IsActive: true, IsEmailVerified: true}},
			wantErr: exceptions.ErrEmailAlreadyVerified,
		},
		{
			name:    "returns ErrAccountInactive when user is inactive",
			repo:    &fakeUserRepo{byID: users.User{IsActive: false}},
			wantErr: exceptions.ErrAccountInactive,
		},
		{
			name:    "returns ErrAccountInactive when user is deleted",
			repo:    &fakeUserRepo{byID: users.User{IsActive: true, IsDeleted: true}},
			wantErr: exceptions.ErrAccountInactive,
		},
		{
			name:    "returns ErrUserNotFound when lock finds no row",
			repo:    &fakeUserRepo{byID: active, lockErr: gorm.ErrRecordNotFound},
			wantErr: exceptions.ErrUserNotFound,
		},
		{
			name:    "returns ErrEmailAlreadyVerified when verified meanwhile",
			repo:    &fakeUserRepo{byID: active, locked: users.User{IsEmailVerified: true}},
			wantErr: exceptions.ErrEmailAlreadyVerified,
		},
		{
			name:     "sends once to the locked email on success",
			repo:     &fakeUserRepo{byID: active, locked: users.User{ID: active.ID, Email: "new@x.com"}, profile: users.UserProfile{FirstName: "Ann"}},
			wantMail: "new@x.com",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &fakeMailer{}
			svc := NewResendVerificationService(tc.repo, mailer, nopLogger())

			err := svc.ResendForUser(context.Background(), uuid.New())

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if len(mailer.sent) != 0 {
					t.Error("mail must not be sent")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mailer.sent) != 1 || mailer.sent[0].to != tc.wantMail || mailer.sent[0].name != "Ann" {
				t.Errorf("mails = %+v, want one to %s", mailer.sent, tc.wantMail)
			}
		})
	}
}

func TestChangeEmailService_ChangeEmail(t *testing.T) {
	const password = "password123"
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	hash, err := cfg.hashStringWithSalt(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	self := users.User{ID: uuid.New(), Email: "old@x.com", PasswordHash: hash, IsActive: true}
	req := ChangeEmailRequest{NewEmail: "  New@X.com ", CurrentPassword: password}

	tests := []struct {
		name        string
		repo        *fakeUserRepo
		checkerErr  error
		req         ChangeEmailRequest
		wantErr     *exceptions.AppError
		wantCode    string
		wantUpdated string
	}{
		{
			name:    "returns ErrUserNotFound when user is missing",
			repo:    &fakeUserRepo{byIDErr: gorm.ErrRecordNotFound},
			req:     req,
			wantErr: exceptions.ErrUserNotFound,
		},
		{
			name:    "returns ErrAccountInactive when user is inactive",
			repo:    &fakeUserRepo{byID: users.User{PasswordHash: hash}},
			req:     req,
			wantErr: exceptions.ErrAccountInactive,
		},
		{
			name:    "returns 409 when user is already verified",
			repo:    &fakeUserRepo{byID: users.User{IsActive: true, IsEmailVerified: true, PasswordHash: hash}},
			req:     req,
			wantErr: exceptions.ErrEmailAlreadyVerified,
		},
		{
			name:    "returns ErrCurrentPasswordInvalid for a wrong password",
			repo:    &fakeUserRepo{byID: self},
			req:     ChangeEmailRequest{NewEmail: "new@x.com", CurrentPassword: "wrong-password"},
			wantErr: exceptions.ErrCurrentPasswordInvalid,
		},
		{
			name:    "returns ErrEmailUnchanged for the same email ignoring case",
			repo:    &fakeUserRepo{byID: self},
			req:     ChangeEmailRequest{NewEmail: " OLD@x.com", CurrentPassword: password},
			wantErr: exceptions.ErrEmailUnchanged,
		},
		{
			name:       "returns typo error from the email checker",
			repo:       &fakeUserRepo{byID: self},
			checkerErr: &emailcheck.TypoError{Suggested: "a@gmail.com"},
			req:        req,
			wantCode:   "EMAIL_DOMAIN_TYPO",
		},
		{
			name: "returns ErrEmailAlreadyInUse when a verified user holds the new email",
			repo: &fakeUserRepo{byID: self, byEmail: map[string]users.User{
				"new@x.com": {ID: uuid.New(), IsEmailVerified: true},
			}},
			req:     req,
			wantErr: exceptions.ErrEmailAlreadyInUse,
		},
		{
			name:    "returns ErrEmailAlreadyVerified when verified meanwhile under lock",
			repo:    &fakeUserRepo{byID: self, locked: users.User{IsEmailVerified: true}},
			req:     req,
			wantErr: exceptions.ErrEmailAlreadyVerified,
		},
		{
			name:    "returns ErrEmailAlreadyVerified when update touches no row",
			repo:    &fakeUserRepo{byID: self, updateEmailOK: false},
			req:     req,
			wantErr: exceptions.ErrEmailAlreadyVerified,
		},
		{
			name:        "updates the email and mails the new address on success",
			repo:        &fakeUserRepo{byID: self, updateEmailOK: true, profile: users.UserProfile{FirstName: "Ann"}},
			req:         req,
			wantUpdated: "new@x.com",
		},
		{
			name: "releases a stale holder of the new email before updating",
			repo: &fakeUserRepo{
				byID: self, updateEmailOK: true, deleted: true,
				byEmail: map[string]users.User{"new@x.com": {ID: uuid.New()}},
			},
			req:         req,
			wantUpdated: "new@x.com",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mailer := &fakeMailer{}
			releaser := NewStaleAccountReleaser(tc.repo, nopLogger())
			svc := NewChangeEmailService(tc.repo, mailer, &fakeEmailChecker{err: tc.checkerErr}, releaser, nopLogger())

			err := svc.ChangeEmail(context.Background(), self.ID, tc.req)

			switch {
			case tc.wantCode != "":
				assertAppCode(t, err, tc.wantCode)
			case tc.wantErr != nil:
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
			case err != nil:
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantUpdated == "" {
				if len(mailer.sent) != 0 {
					t.Errorf("mail must not be sent, got %+v", mailer.sent)
				}
				return
			}
			if tc.repo.updatedEmail != tc.wantUpdated {
				t.Errorf("updated email = %q, want %q", tc.repo.updatedEmail, tc.wantUpdated)
			}
			if len(mailer.sent) != 1 || mailer.sent[0].to != tc.wantUpdated {
				t.Errorf("mails = %+v, want one to %s", mailer.sent, tc.wantUpdated)
			}
		})
	}
}

func TestLoginService_ValidateUserDetail(t *testing.T) {
	tests := []struct {
		name     string
		user     users.User
		wantCode string
	}{
		{name: "allows an active unverified user", user: users.User{IsActive: true}},
		{name: "allows an active verified user", user: users.User{IsActive: true, IsEmailVerified: true}},
		{name: "blocks a suspended user", user: users.User{IsActive: false}, wantCode: "ACCOUNT_SUSPENDED"},
		{name: "blocks a deleted user", user: users.User{IsActive: true, IsDeleted: true}, wantCode: "ACCOUNT_DELETED"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &LoginService{logger: nopLogger()}

			err := svc.ValidateUserDetail(&tc.user)

			if tc.wantCode != "" {
				assertAppCode(t, err, tc.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
