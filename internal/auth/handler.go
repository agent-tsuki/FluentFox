// Package auth — handler.go.
// Owns HTTP concerns only: parse request, validate, call service, write response.
// Must never contain SQL, business logic, or direct imports of pgx.
package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/validator"
	"go.uber.org/zap"
)

// Handler holds the auth service and dependencies needed by auth HTTP handlers.
type Handler struct {
	svc       *Service
	validator *validator.Validator
	log       *zap.Logger
}

// NewHandler constructs an auth Handler.
func NewHandler(svc *Service, v *validator.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: v, log: log}
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			response.Conflict(w, "an account with this email already exists")
			return
		}
		h.log.Error("auth: register", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSON(w, http.StatusCreated, RegisterResponse{
		Message: "account created — please verify your email",
		UserID:  userID.String(),
	})
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	tokens, err := h.svc.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Unauthorized(w, "invalid email or password")
			return
		}
		h.log.Error("auth: login", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	tokens, err := h.svc.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(w, "invalid or expired refresh token")
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	_ = h.svc.Logout(r.Context(), req.RefreshToken)
	response.JSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// ForgotPassword handles POST /auth/forgot-password.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	// Always 200 — no enumeration of whether the email exists.
	_ = h.svc.ForgotPassword(r.Context(), req.Email)
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "if an account exists for that email, a reset link has been sent",
	})
}

// ResetPassword handles POST /auth/reset-password.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	err := h.svc.ResetPassword(r.Context(), req.Token, req.Password)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) || errors.Is(err, ErrTokenAlreadyUsed) {
			response.BadRequest(w, "reset token is invalid, expired, or already used")
			return
		}
		h.log.Error("auth: reset password", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

// VerifyEmail handles POST /auth/verify-email.
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	err := h.svc.VerifyEmail(r.Context(), req.Token)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) || errors.Is(err, ErrTokenAlreadyUsed) {
			response.BadRequest(w, "verification token is invalid, expired, or already used")
			return
		}
		h.log.Error("auth: verify email", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "email verified successfully"})
}
