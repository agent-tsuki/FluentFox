// Package auth — model.go.
// Defines DB models, request DTOs, and response DTOs for authentication.
// These are three separate sets of types — they must never be conflated.
// Password fields NEVER appear on any response DTO.
package auth

import (
	"time"

	"github.com/google/uuid"
)

// --- DB models (map exactly to table columns) ---

// RefreshToken is the DB row for the refresh_tokens table.
type RefreshToken struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"` // SHA-256 of the raw token — never the raw value
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
	RevokedAt *time.Time `db:"revoked_at"`
}

// EmailVerification is the DB row for the email_verifications table.
type EmailVerification struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time `db:"created_at"`
}

// PasswordReset is the DB row for the password_resets table.
type PasswordReset struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time `db:"created_at"`
}

// --- Request DTOs (bound from JSON body by handlers) ---

// RegisterRequest is the payload for POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	Username string `json:"username" validate:"required,min=3,max=30"`
}

// LoginRequest is the payload for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest is the payload for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ForgotPasswordRequest is the payload for POST /auth/forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest is the payload for POST /auth/reset-password.
type ResetPasswordRequest struct {
	Token    string `json:"token"    validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// VerifyEmailRequest is the payload for POST /auth/verify-email.
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// --- Response DTOs (written to JSON by handlers) ---

// TokenResponse is returned after successful login or token refresh.
// It never includes the password hash or raw refresh token stored in DB.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"` // raw opaque value sent to client
	ExpiresIn    int    `json:"expires_in"`    // seconds until access token expires
}

// RegisterResponse is returned after successful registration.
type RegisterResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}
