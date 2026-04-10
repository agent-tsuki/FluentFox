package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/validator"
	"go.uber.org/zap"
)

type Handler struct {
	authService   *AuthService
	verifyService *TokenVerificationService
	logger        *zap.Logger
	validate      *validator.Validator
}

func NewHandler(authService *AuthService, verifyService *TokenVerificationService, log *zap.Logger, v *validator.Validator) *Handler {
	return &Handler{
		authService:   authService,
		verifyService: verifyService,
		logger:        log,
		validate:      v,
	}
}

// POST /auth/register
func (h *Handler) AuthRegister(c *gin.Context) {
	log := middleware.LoggerFromContext(c.Request.Context(), h.logger)

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.validate.Validate(req); err != nil {
		response.HandleError(c.Writer, err, log)
		return
	}
	if err := h.authService.registerUser(c.Request.Context(), req); err != nil {
		response.HandleError(c.Writer, err, log)
		return
	}
	response.JSON(c.Writer, http.StatusCreated, map[string]string{
		"message": "registration successful, check your email to verify",
	})
}

// POST /auth/verify?token=<token>
func (h *Handler) AuthVerify(c *gin.Context) {
	log := middleware.LoggerFromContext(c.Request.Context(), h.logger)

	token := c.Query("token")
	if token == "" {
		response.BadRequest(c.Writer, "token is required")
		return
	}
	if err := h.verifyService.VerifyUserToken(c.Request.Context(), token); err != nil {
		response.HandleError(c.Writer, err, log)
		return
	}
	response.JSON(c.Writer, http.StatusOK, map[string]string{
		"message": "email verified successfully",
	})
}
