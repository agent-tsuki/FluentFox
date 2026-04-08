package auth

import (
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/exceptions/exception"
	"github.com/fluentfox/api/pkg/response"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Register godoc
// @Summary     Register a new user
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body RegisterRequest true "Register"
// @Success     201  {object} AuthResponse
// @Router      /api/auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request, c *gin.Context) {
	var req RegisterRequest
	if err := response.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, exception.ErrEmailAlreadyExists):
			response.Error(w, http.StatusConflict, err.Error())
		default:
			h.log.Error("register failed", zap.Error(err))
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

// Login godoc
// @Summary     Login
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body LoginRequest true "Login"
// @Success     200  {object} AuthResponse
// @Router      /api/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := response.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, exception.ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, err.Error())
		case errors.Is(err, exception.ErrAccountInactive):
			response.Error(w, http.StatusForbidden, err.Error())
		default:
			h.log.Error("login failed", zap.Error(err))
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// Me godoc
// @Summary     Get current user
// @Tags        auth
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} UserResponse
// @Router      /api/auth/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := token.UserIDFromContext(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.Me(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, exception.ErrUserNotFound):
			response.Error(w, http.StatusNotFound, err.Error())
		default:
			h.log.Error("me failed", zap.Error(err))
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, user)
}