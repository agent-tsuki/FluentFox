// Package middleware — auth.go.
// RequireAuth validates the JWT in the Authorization header and injects
// the parsed Claims into the request context. Downstream handlers and services
// retrieve the authenticated user via ContextUserID / ContextUserRole.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/token"
	"github.com/google/uuid"
)

// RequireAuth returns a middleware that enforces a valid JWT access token.
// It reads the token from the "Authorization: Bearer <token>" header.
// On success it injects the user_id and role into the request context.
// On failure it writes 401 and stops the chain.
func RequireAuth(maker *token.Maker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Unauthorized(w, "missing or malformed Authorization header")
				return
			}

			raw := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := maker.ValidateAccessToken(raw)
			if err != nil {
				response.Unauthorized(w, "invalid or expired access token")
				return
			}

			ctx := context.WithValue(r.Context(), contextUserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, contextUserRoleKey, claims.Role)
			ctx = context.WithValue(ctx, contextEmailVerifiedKey, claims.EmailVerified)

			// Enrich the request-scoped logger with user_id.
			if l, ok := ctx.Value(loggerKey).(interface {
				With(...interface{}) interface{}
			}); ok {
				_ = l
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireEmailVerified is a middleware that rejects requests from users
// whose email address has not been verified yet.
func RequireEmailVerified(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ContextEmailVerified(r.Context()) {
			response.Forbidden(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type authContextKey string

const (
	contextUserIDKey       authContextKey = "auth_user_id"
	contextUserRoleKey     authContextKey = "auth_user_role"
	contextEmailVerifiedKey authContextKey = "auth_email_verified"
)

// ContextUserID extracts the authenticated user ID from the context.
// Returns uuid.Nil if the user is not authenticated.
func ContextUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(contextUserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// ContextUserRole extracts the authenticated user role from the context.
func ContextUserRole(ctx context.Context) string {
	if role, ok := ctx.Value(contextUserRoleKey).(string); ok {
		return role
	}
	return ""
}

// ContextEmailVerified reports whether the authenticated user has verified their email.
func ContextEmailVerified(ctx context.Context) bool {
	if v, ok := ctx.Value(contextEmailVerifiedKey).(bool); ok {
		return v
	}
	return false
}
