// Package middleware — rbac.go.
// RequireRole enforces role-based access control after RequireAuth has run.
// It reads the role from the context (set by auth middleware) and rejects
// requests whose role is not in the allowed set.
package middleware

import (
	"net/http"

	"github.com/fluentfox/api/pkg/response"
)

// RequireRole returns a middleware that allows only requests from users
// whose role is in the provided allowedRoles list.
// RequireAuth must run before this middleware.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := ContextUserRole(r.Context())
			if _, ok := allowed[role]; !ok {
				response.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
