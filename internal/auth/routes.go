package auth

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts auth endpoints onto r.
// All dependencies are injected via h — construction happens in main.
func RegisterRoutes(r gin.IRouter, h *Handler) {
	auth := r.Group("/auth")
	auth.POST("/register", h.AuthRegister)
	auth.POST("/verify", h.AuthVerify)
}
