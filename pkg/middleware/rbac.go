// Package middleware — rbac.go
// RequireRole enforces role-based access after RequireAuth has run.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/fluentfox/api/pkg/response"
)

// RequireRole allows only requests whose role is in allowedRoles.
// RequireAuth must run before this middleware.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role := ContextUserRole(c.Request.Context())
		if _, ok := allowed[role]; !ok {
			response.Forbidden(c.Writer)
			c.Abort()
			return
		}
		c.Next()
	}
}
