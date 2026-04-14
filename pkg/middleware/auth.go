// Package middleware — auth.go
// RequireAuth validates the JWT in the Authorization header and injects
// parsed Claims into both the Gin context and the request context so
// downstream services can read them via ContextUserID etc.
package middleware

import (
	"context"
	"strings"

	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type authContextKey string

const (
	contextUserIDKey        authContextKey = "auth_user_id"
	contextIsAdminKey       authContextKey = "auth_is_admin"
	contextEmailVerifiedKey authContextKey = "auth_email_verified"
)

// RequireAuth enforces a valid JWT access token.
func RequireAuth(maker *token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Unauthorized(c.Writer, "missing or malformed Authorization header")
			c.Abort()
			return
		}

		raw := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := maker.ValidateAccessToken(raw)
		if err != nil {
			response.Unauthorized(c.Writer, "invalid or expired access token")
			c.Abort()
			return
		}

		// Inject into stdlib context so services can read via ContextUserID(ctx).
		ctx := context.WithValue(c.Request.Context(), contextUserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, contextIsAdminKey, claims.IsAdmin)
		ctx = context.WithValue(ctx, contextEmailVerifiedKey, claims.EmailVerified)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequireEmailVerified rejects requests from users whose email is not verified.
// Must run after RequireAuth.
func RequireEmailVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ContextEmailVerified(c.Request.Context()) {
			response.Forbidden(c.Writer)
			c.Abort()
			return
		}
		c.Next()
	}
}

// ContextUserID extracts the authenticated user ID from the context.
func ContextUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(contextUserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// ContextIsAdmin reports whether the authenticated user is an admin.
func ContextIsAdmin(ctx context.Context) bool {
	if v, ok := ctx.Value(contextIsAdminKey).(bool); ok {
		return v
	}
	return false
}

// ContextEmailVerified reports whether the authenticated user has verified their email.
func ContextEmailVerified(ctx context.Context) bool {
	if v, ok := ctx.Value(contextEmailVerifiedKey).(bool); ok {
		return v
	}
	return false
}
