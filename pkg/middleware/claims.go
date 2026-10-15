package middleware

import (
	"context"

	"github.com/fluentfox/api/pkg/token"
)

// Same keys as RequireAuth, so ContextUserID works for huma routes too.
func ContextWithClaims(ctx context.Context, claims *token.Claims) context.Context {
	ctx = context.WithValue(ctx, contextUserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, contextIsAdminKey, claims.IsAdmin)
	return context.WithValue(ctx, contextEmailVerifiedKey, claims.EmailVerified)
}
