package token

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func signWith(t *testing.T, method jwt.SigningMethod, key any, claims Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestMaker_AccessToken_roundTrip(t *testing.T) {
	m := NewMaker(testSecret, 15, 7)
	uid := uuid.New()

	raw, err := m.GenerateAccessToken(uid, true, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.ValidateAccessToken(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != uid || !got.IsAdmin || !got.EmailVerified {
		t.Fatalf("claims did not round trip: %+v", got)
	}
}

func TestMaker_ValidateAccessToken_rejects(t *testing.T) {
	m := NewMaker(testSecret, 15, 7)
	exp := jwt.NewNumericDate(time.Now().Add(time.Hour))
	good := func() Claims {
		return Claims{UserID: uuid.New(), RegisteredClaims: jwt.RegisteredClaims{
			Issuer: jwtIssuer, Audience: jwt.ClaimStrings{jwtAudience}, ExpiresAt: exp,
		}}
	}
	mod := func(f func(*Claims)) Claims { c := good(); f(&c); return c }
	hs := jwt.SigningMethodHS256
	secret := []byte(testSecret)

	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, good()).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{"returns ErrTokenExpired when token is past expiry", signWith(t, hs, secret, mod(func(c *Claims) {
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
		})), ErrTokenExpired},
		{"rejects a token with no issuer or audience", signWith(t, hs, secret, mod(func(c *Claims) {
			c.Issuer, c.Audience = "", nil
		})), jwt.ErrTokenRequiredClaimMissing},
		{"rejects a token with the wrong audience", signWith(t, hs, secret, mod(func(c *Claims) {
			c.Audience = jwt.ClaimStrings{"other-app"}
		})), jwt.ErrTokenInvalidAudience},
		{"rejects a token with the wrong issuer", signWith(t, hs, secret, mod(func(c *Claims) {
			c.Issuer = "other-service"
		})), jwt.ErrTokenInvalidIssuer},
		{"rejects a token with no expiry", signWith(t, hs, secret, mod(func(c *Claims) {
			c.ExpiresAt = nil
		})), jwt.ErrTokenInvalidClaims},
		{"rejects a token signed with another secret", signWith(t, hs, []byte("another-secret-another-secret-1234"), good()), jwt.ErrTokenSignatureInvalid},
		{"rejects a token with alg none", noneToken, jwt.ErrTokenUnverifiable},
		{"rejects garbage", "not-a-jwt", jwt.ErrTokenMalformed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.ValidateAccessToken(tc.raw)
			if err == nil {
				t.Fatalf("expected error, got claims %+v", got)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}
