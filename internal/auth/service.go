// Package auth — service.go.
// Owns all authentication business logic: registration, login, token refresh,
// email verification, and password reset flows.
// Must never know about HTTP — no http.Request, no http.ResponseWriter.
// Must never write SQL — all DB access goes through Repository.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fluentfox/api/internal/user"
	"github.com/fluentfox/api/pkg/mailer"
	"github.com/fluentfox/api/pkg/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials is returned when email/password do not match.
var ErrInvalidCredentials = errors.New("auth service: invalid email or password")

// ErrTokenExpired is returned when a verification or reset token has expired.
var ErrTokenExpired = errors.New("auth service: token has expired")

// ErrTokenAlreadyUsed is returned when a one-time token has already been consumed.
var ErrTokenAlreadyUsed = errors.New("auth service: token has already been used")

// ErrEmailAlreadyExists is returned when the email is already registered.
var ErrEmailAlreadyExists = errors.New("auth service: email already registered")

// Service handles auth business logic.
type Service struct {
	authRepo *Repository
	userRepo *user.Repository
	maker    *token.Maker
	mailer   mailer.Mailer
	appURL   string
	refreshExpiryDays int
}

// NewService constructs an auth Service.
func NewService(
	authRepo *Repository,
	userRepo *user.Repository,
	maker *token.Maker,
	mailer mailer.Mailer,
	appURL string,
	refreshExpiryDays int,
) *Service {
	return &Service{
		authRepo:          authRepo,
		userRepo:          userRepo,
		maker:             maker,
		mailer:            mailer,
		appURL:            appURL,
		refreshExpiryDays: refreshExpiryDays,
	}
}

// Register creates a new user account, hashes the password, and sends
// a verification email. Returns the new user ID on success.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (uuid.UUID, error) {
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return uuid.Nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, fmt.Errorf("auth service: hash password: %w", err)
	}

	newUser, err := s.userRepo.Create(ctx, req.Email, req.Username, string(hash))
	if err != nil {
		return uuid.Nil, fmt.Errorf("auth service: create user: %w", err)
	}

	rawToken, err := s.maker.GenerateRefreshToken()
	if err != nil {
		return uuid.Nil, fmt.Errorf("auth service: generate verification token: %w", err)
	}

	tokenHash := token.HashToken(rawToken)
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.authRepo.CreateEmailVerification(ctx, newUser.ID, tokenHash, expiresAt); err != nil {
		return uuid.Nil, fmt.Errorf("auth service: store verification token: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, rawToken)
	if err := s.mailer.SendVerificationEmail(ctx, newUser.Email, newUser.Username, verifyURL); err != nil {
		// Non-fatal: user can request a new verification email.
		// Log this at the call site.
		_ = err
	}

	return newUser.ID, nil
}

// Login authenticates a user and returns a token pair on success.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	u, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, u.ID, u.Role, u.EmailVerified)
}

// RefreshTokens validates a refresh token and issues a new token pair.
// The old refresh token is revoked (rotation) — each token is single-use.
func (s *Service) RefreshTokens(ctx context.Context, rawRefreshToken string) (*TokenResponse, error) {
	hash := token.HashToken(rawRefreshToken)

	rt, err := s.authRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("auth service: refresh token not found: %w", err)
	}

	if rt.RevokedAt != nil {
		return nil, fmt.Errorf("auth service: refresh token revoked")
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	if err := s.authRepo.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("auth service: revoke old refresh token: %w", err)
	}

	u, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth service: get user for refresh: %w", err)
	}

	return s.issueTokenPair(ctx, u.ID, u.Role, u.EmailVerified)
}

// Logout revokes the given refresh token.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := token.HashToken(rawRefreshToken)
	rt, err := s.authRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("auth service: logout — token not found: %w", err)
	}
	return s.authRepo.RevokeRefreshToken(ctx, rt.ID)
}

// ForgotPassword creates a password reset token and sends the reset email.
// Always returns nil to avoid user enumeration — even if email not found.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil // silent — no enumeration
	}

	rawToken, err := s.maker.GenerateRefreshToken()
	if err != nil {
		return fmt.Errorf("auth service: generate reset token: %w", err)
	}

	tokenHash := token.HashToken(rawToken)
	expiresAt := time.Now().Add(1 * time.Hour)
	if err := s.authRepo.CreatePasswordReset(ctx, u.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("auth service: store reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, rawToken)
	return s.mailer.SendPasswordReset(ctx, u.Email, u.Username, resetURL)
}

// ResetPassword validates the reset token, hashes the new password, and updates the user.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	hash := token.HashToken(rawToken)

	pr, err := s.authRepo.GetPasswordResetByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("auth service: reset token not found: %w", err)
	}

	if pr.UsedAt != nil {
		return ErrTokenAlreadyUsed
	}

	if time.Now().After(pr.ExpiresAt) {
		return ErrTokenExpired
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth service: hash new password: %w", err)
	}

	if err := s.userRepo.UpdatePasswordHash(ctx, pr.UserID, string(newHash)); err != nil {
		return fmt.Errorf("auth service: update password: %w", err)
	}

	if err := s.authRepo.MarkPasswordResetUsed(ctx, pr.ID); err != nil {
		return fmt.Errorf("auth service: mark reset used: %w", err)
	}

	return s.authRepo.RevokeAllUserRefreshTokens(ctx, pr.UserID)
}

// VerifyEmail marks the user's email as verified.
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	hash := token.HashToken(rawToken)

	ev, err := s.authRepo.GetEmailVerificationByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("auth service: verification token not found: %w", err)
	}

	if ev.UsedAt != nil {
		return ErrTokenAlreadyUsed
	}

	if time.Now().After(ev.ExpiresAt) {
		return ErrTokenExpired
	}

	if err := s.userRepo.SetEmailVerified(ctx, ev.UserID); err != nil {
		return fmt.Errorf("auth service: set email verified: %w", err)
	}

	return s.authRepo.MarkEmailVerificationUsed(ctx, ev.ID)
}

// issueTokenPair creates a new access+refresh token pair and stores the refresh token.
func (s *Service) issueTokenPair(ctx context.Context, userID uuid.UUID, role string, emailVerified bool) (*TokenResponse, error) {
	accessToken, err := s.maker.GenerateAccessToken(userID, role, emailVerified)
	if err != nil {
		return nil, fmt.Errorf("auth service: generate access token: %w", err)
	}

	rawRefresh, err := s.maker.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("auth service: generate refresh token: %w", err)
	}

	refreshHash := token.HashToken(rawRefresh)
	expiresAt := time.Now().Add(time.Duration(s.refreshExpiryDays) * 24 * time.Hour)

	if _, err := s.authRepo.CreateRefreshToken(ctx, userID, refreshHash, expiresAt); err != nil {
		return nil, fmt.Errorf("auth service: store refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    3600,
	}, nil
}
