// Package middleware — rbac.go
// RequireAdmin enforces admin-only access after RequireAuth has run.
package middleware

import (
	"github.com/fluentfox/api/pkg/response"
	"github.com/gin-gonic/gin"
)

// RequireAdmin allows only requests from admin users.
// RequireAuth must run before this middleware.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ContextIsAdmin(c.Request.Context()) {
			response.Forbidden(c.Writer)
			c.Abort()
			return
		}
		c.Next()
	}
}
