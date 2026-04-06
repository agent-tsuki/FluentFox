// Package token handles JWT generation and validation for the auth system.
// It never reads configuration from the environment directly — secrets and
// expiry values are passed in by the caller.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the payload embedded in every access token.
type Claims struct {
	UserID        uuid.UUID `json:"user_id"`
	Role          string    `json:"role"`
	EmailVerified bool      `json:"email_verified"`
	jwt.RegisteredClaims
}

// Maker holds the secrets needed to sign and verify tokens.
// Create one per application run; do not create per request.
type Maker struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewMaker constructs a Maker. accessExpiryMinutes and refreshExpiryDays come
// from the Config struct — never from env vars read here.
func NewMaker(accessSecret, refreshSecret string, accessExpiryMinutes, refreshExpiryDays int) *Maker {
	return &Maker{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  time.Duration(accessExpiryMinutes) * time.Minute,
		refreshExpiry: time.Duration(refreshExpiryDays) * 24 * time.Hour,
	}
}

// GenerateAccessToken creates a signed JWT access token for the given user.
// The token carries user_id, role, and email_verified in its claims.
func (m *Maker) GenerateAccessToken(userID uuid.UUID, role string, emailVerified bool) (string, error) {
	claims := Claims{
		UserID:        userID,
		Role:          role,
		EmailVerified: emailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.accessSecret)
	if err != nil {
		return "", fmt.Errorf("token: sign access token: %w", err)
	}
	return signed, nil
}

// GenerateRefreshToken returns a random opaque string intended for storage
// in the database. It is NOT a JWT — use HashToken before persisting it.
func (m *Maker) GenerateRefreshToken() (string, error) {
	return uuid.NewString(), nil
}

// ValidateAccessToken parses and validates a raw JWT string.
// Returns Claims on success or an error if expired, tampered, or malformed.
func (m *Maker) ValidateAccessToken(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("token: unexpected signing method: %v", t.Header["alg"])
		}
		return m.accessSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("token: validate access token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ErrTokenExpired is returned when a token's expiry has passed.
var ErrTokenExpired = errors.New("token: access token has expired")

// ErrTokenInvalid is returned when a token cannot be parsed or verified.
var ErrTokenInvalid = errors.New("token: access token is invalid")
