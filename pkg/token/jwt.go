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
	IsAdmin          bool    `json:"is_admin"`
	EmailVerified bool      `json:"email_verified"`
	// The refresh_tokens row this token was issued for, so the device list can
	// mark "this device". Nil in tokens issued before sessions existed.
	SessionID uuid.UUID `json:"sid"`
	jwt.RegisteredClaims
}

// Issuer and audience stop a token signed for another service that happens to
// share the secret from being accepted here.
const (
	jwtIssuer   = "fluentfox-api"
	jwtAudience = "fluentfox-app"
)

type Maker struct {
	accessSecret  []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewMaker(accessSecret string, accessExpiryMinutes, refreshExpiryDays int) *Maker {
	return &Maker{
		accessSecret:  []byte(accessSecret),
		accessExpiry:  time.Duration(accessExpiryMinutes) * time.Minute,
		refreshExpiry: time.Duration(refreshExpiryDays) * 24 * time.Hour,
	}
}

func (m *Maker) GenerateAccessToken(userID uuid.UUID, is_admin bool, emailVerified bool, sessionID uuid.UUID) (string, error) {
	claims := Claims{
		UserID:        userID,
		IsAdmin:          is_admin,
		EmailVerified: emailVerified,
		SessionID:     sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Audience:  jwt.ClaimStrings{jwtAudience},
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

func (m *Maker) GenerateRefreshToken() (string, error) {
	return uuid.NewString(), nil
}

// RefreshExpiryTime returns the absolute time at which a refresh token issued
// now should expire.
func (m *Maker) RefreshExpiryTime() time.Time {
	return time.Now().UTC().Add(m.refreshExpiry)
}

func (m *Maker) ValidateAccessToken(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("token: unexpected signing method: %v", t.Header["alg"])
		}
		return m.accessSecret, nil
	}, jwt.WithIssuer(jwtIssuer), jwt.WithAudience(jwtAudience), jwt.WithExpirationRequired())
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
