package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/token"
)

const loginTestPassword = "correct-password"

func newTestMaker() *token.Maker {
	return token.NewMaker("0123456789abcdef0123456789abcdef", 15, 7)
}

func mustHash(t *testing.T, plain string) string {
	t.Helper()
	cfg := Argon2Config{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	h, err := cfg.hashStringWithSalt(plain)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestLoginService_Login(t *testing.T) {
	hash := mustHash(t, loginTestPassword)
	base := users.User{ID: uuid.New(), Email: "john@example.com", Username: "john", PasswordHash: hash, IsActive: true, IsEmailVerified: true}
	with := func(mod func(*users.User)) users.User {
		u := base
		mod(&u)
		return u
	}
	dbErr := errors.New("connection refused")

	tests := []struct {
		name        string
		repo        *fakeUserRepo
		email, pass string
		wantErr     error
		wantNonApp  bool
		wantLookup  string
		wantTokens  bool
	}{
		{
			name:  "returns ErrInvalidCredentials for an unknown email",
			repo:  &fakeUserRepo{emailErr: fmt.Errorf("x: %w", gorm.ErrRecordNotFound)},
			email: "john@example.com", pass: loginTestPassword,
			wantErr: exceptions.ErrInvalidCredentials,
		},
		{
			name:  "returns ErrInvalidCredentials for a wrong password",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": base}},
			email: "john@example.com", pass: "wrong-password",
			wantErr: exceptions.ErrInvalidCredentials,
		},
		{
			name:  "returns ErrInvalidCredentials, not a status error, for a suspended user with a wrong password",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": with(func(u *users.User) { u.IsActive = false })}},
			email: "john@example.com", pass: "wrong-password",
			wantErr: exceptions.ErrInvalidCredentials,
		},
		{
			name:  "returns ErrInvalidCredentials, not a status error, for a deleted user with a wrong password",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": with(func(u *users.User) { u.IsDeleted = true })}},
			email: "john@example.com", pass: "wrong-password",
			wantErr: exceptions.ErrInvalidCredentials,
		},
		{
			name:  "returns ErrAccountSuspended for a suspended user with the correct password",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": with(func(u *users.User) { u.IsActive = false })}},
			email: "john@example.com", pass: loginTestPassword,
			wantErr: exceptions.ErrAccountSuspended,
		},
		{
			name:  "returns ErrAccountDeleted for a deleted user with the correct password",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": with(func(u *users.User) { u.IsDeleted = true })}},
			email: "john@example.com", pass: loginTestPassword,
			wantErr: exceptions.ErrAccountDeleted,
		},
		{
			name:  "returns a non-AppError wrapping the cause on a database error",
			repo:  &fakeUserRepo{emailErr: dbErr},
			email: "john@example.com", pass: loginTestPassword,
			wantErr:    dbErr,
			wantNonApp: true,
		},
		{
			name:  "returns a non-AppError when the stored hash is malformed",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": with(func(u *users.User) { u.PasswordHash = "garbage" })}},
			email: "john@example.com", pass: loginTestPassword,
			wantErr:    errMalformedPasswordHash,
			wantNonApp: true,
		},
		{
			name:  "returns a non-AppError when storing the session fails",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": base}, sessionErr: dbErr},
			email: "john@example.com", pass: loginTestPassword,
			wantErr:    dbErr,
			wantNonApp: true,
		},
		{
			name:  "normalizes the email before lookup",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": base}},
			email: "  John@Example.COM ", pass: loginTestPassword,
			wantLookup: "john@example.com",
			wantTokens: true,
		},
		{
			name:  "logs in an unverified active user with the correct password",
			repo:  &fakeUserRepo{byEmail: map[string]users.User{"john@example.com": with(func(u *users.User) { u.IsEmailVerified = false })}},
			email: "john@example.com", pass: loginTestPassword,
			wantTokens: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewLogin(tc.repo, newTestMaker(), nil, nil, nopLogger())
			resp, err := svc.Login(context.Background(), LoginRequest{Email: tc.email, Password: tc.pass})

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				if _, isApp := exceptions.As(err); isApp == tc.wantNonApp {
					t.Fatalf("AppError = %v, want %v", isApp, !tc.wantNonApp)
				}
				if resp.AccessToken != "" || resp.RefreshToken != "" {
					t.Fatal("tokens must not be issued on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantLookup != "" && tc.repo.lookedUpEmail != tc.wantLookup {
				t.Fatalf("looked up %q, want %q", tc.repo.lookedUpEmail, tc.wantLookup)
			}
			if tc.wantTokens {
				assertIssuedTokens(t, tc.repo, resp)
			}
		})
	}
}

func assertIssuedTokens(t *testing.T, repo *fakeUserRepo, resp LoginResponse) {
	t.Helper()
	if repo.createdSessions != 1 {
		t.Fatalf("CreateSession called %d times, want 1", repo.createdSessions)
	}
	if !repo.lastLoginSet {
		t.Fatal("last login was not recorded")
	}
	assertTokenPair(t, resp, repo.createdSessionHash, repo.createdSessionID)
}

// The raw refresh token must never be stored, and the access token must name
// the session it belongs to.
func assertTokenPair(t *testing.T, resp LoginResponse, storedHash string, wantSessionID uuid.UUID) {
	t.Helper()
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}
	if storedHash == resp.RefreshToken {
		t.Fatal("raw refresh token was stored")
	}
	if storedHash != hashVerificationToken(resp.RefreshToken) {
		t.Fatal("stored value is not the hash of the refresh token")
	}
	claims, err := newTestMaker().ValidateAccessToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("access token invalid: %v", err)
	}
	if claims.SessionID != wantSessionID {
		t.Fatalf("sid = %s, want %s", claims.SessionID, wantSessionID)
	}
}

func TestLoginService_RefreshToken(t *testing.T) {
	uid := uuid.New()
	valid := users.RefreshToken{ID: uuid.New(), UserID: uid, TokenHash: "stored-hash", ExpiresAt: time.Now().UTC().Add(time.Hour)}
	activeUser := users.User{ID: uid, IsActive: true, IsEmailVerified: true}
	dbErr := errors.New("connection refused")

	tests := []struct {
		name       string
		repo       *fakeUserRepo
		raw        string
		wantErr    error
		wantNonApp bool
	}{
		{"returns ErrBadRequest for an empty token", &fakeUserRepo{}, "", exceptions.ErrBadRequest, false},
		{"returns ErrTokenInvalid when the token is not found", &fakeUserRepo{refreshErr: fmt.Errorf("x: %w", gorm.ErrRecordNotFound)}, "raw", exceptions.ErrTokenInvalid, false},
		{"returns a non-AppError on a token lookup database error", &fakeUserRepo{refreshErr: dbErr}, "raw", dbErr, true},
		{"returns ErrTokenInvalid when the token is revoked", &fakeUserRepo{refresh: users.RefreshToken{UserID: uid, IsRevoked: true, ExpiresAt: valid.ExpiresAt}}, "raw", exceptions.ErrTokenInvalid, false},
		{"returns ErrTokenExpired when the token is past expiry", &fakeUserRepo{refresh: users.RefreshToken{UserID: uid, ExpiresAt: time.Now().UTC().Add(-time.Minute)}}, "raw", exceptions.ErrTokenExpired, false},
		{"returns ErrUserNotFound when the user no longer exists", &fakeUserRepo{refresh: valid, byIDErr: fmt.Errorf("x: %w", gorm.ErrRecordNotFound)}, "raw", exceptions.ErrUserNotFound, false},
		{"returns a non-AppError on a user lookup database error", &fakeUserRepo{refresh: valid, byIDErr: dbErr}, "raw", dbErr, true},
		{"returns ErrAccountSuspended when the user is suspended", &fakeUserRepo{refresh: valid, byID: users.User{ID: uid}}, "raw", exceptions.ErrAccountSuspended, false},
		{"returns a non-AppError when storing the rotated token fails", &fakeUserRepo{refresh: valid, byID: activeUser, rotateErr: dbErr}, "raw", dbErr, true},
		{"returns ErrTokenInvalid when a concurrent refresh already rotated the token", &fakeUserRepo{refresh: valid, byID: activeUser, rotateLost: true}, "raw", exceptions.ErrTokenInvalid, false},
		{"rotates the token in place and keeps the session id", &fakeUserRepo{refresh: valid, byID: activeUser}, "raw", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewLogin(tc.repo, newTestMaker(), nil, nil, nopLogger())
			resp, err := svc.RefreshToken(context.Background(), tc.raw)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.repo.createdSessions != 0 {
					t.Fatal("refresh must not create a new session")
				}
				assertTokenPair(t, resp, tc.repo.rotatedHash, valid.ID)
				if resp.RefreshToken == tc.raw {
					t.Fatal("refresh token must rotate")
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
			if _, isApp := exceptions.As(err); isApp == tc.wantNonApp {
				t.Fatalf("AppError = %v, want %v", isApp, !tc.wantNonApp)
			}
		})
	}
}

func TestLoginService_Logout(t *testing.T) {
	uid := uuid.New()
	sessionID := uuid.New()
	dbErr := errors.New("connection refused")

	tests := []struct {
		name        string
		repo        *fakeUserRepo
		raw         string
		wantErr     error
		wantNonApp  bool
		wantRevoked bool
	}{
		{"returns ErrBadRequest for an empty token", &fakeUserRepo{}, "", exceptions.ErrBadRequest, false, false},
		{"returns ErrTokenInvalid when the token is not found", &fakeUserRepo{refreshErr: fmt.Errorf("x: %w", gorm.ErrRecordNotFound)}, "raw", exceptions.ErrTokenInvalid, false, false},
		{"returns a non-AppError on a token lookup database error", &fakeUserRepo{refreshErr: dbErr}, "raw", dbErr, true, false},
		{"returns a non-AppError when revoking fails", &fakeUserRepo{refresh: users.RefreshToken{ID: sessionID, UserID: uid}, revokeErr: dbErr}, "raw", dbErr, true, true},
		{"revokes only this device's session on success", &fakeUserRepo{refresh: users.RefreshToken{ID: sessionID, UserID: uid}}, "raw", nil, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewLogin(tc.repo, newTestMaker(), nil, nil, nopLogger())
			err := svc.Logout(context.Background(), tc.raw)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				if _, isApp := exceptions.As(err); isApp == tc.wantNonApp {
					t.Fatalf("AppError = %v, want %v", isApp, !tc.wantNonApp)
				}
			}
			if got := len(tc.repo.revokedSessionIDs) == 1 && tc.repo.revokedSessionIDs[0] == sessionID; got != tc.wantRevoked {
				t.Fatalf("revoked = %v, want %v", got, tc.wantRevoked)
			}
		})
	}
}
