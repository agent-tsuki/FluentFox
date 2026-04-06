// Package user — handler.go.
// HTTP handlers for user self-service routes: profile, settings, password change.
// Owns HTTP concerns only. Never SQL. Never business logic.
package user

import (
	"encoding/json"
	"net/http"

	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/validator"
	"go.uber.org/zap"
)

// Handler holds dependencies for user HTTP handlers.
type Handler struct {
	svc       *Service
	validator *validator.Validator
	log       *zap.Logger
}

// NewHandler constructs a user Handler.
func NewHandler(svc *Service, v *validator.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: v, log: log}
}

// GetMe godoc
// @Summary      Get current user
// @Description  Returns the authenticated user's account details.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} UserResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /users/me [get]
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	user, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		h.log.Error("user: get me", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, user)
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Returns the authenticated user's public profile (display name, bio, JLPT goal).
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} ProfileResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /users/me/profile [get]
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	profile, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		h.log.Error("user: get profile", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, profile)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Updates display name, bio, native language, and/or JLPT goal.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body UpdateProfileRequest true "Profile fields to update (all optional)"
// @Success      200 {object} map[string]string
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      422 {object} response.ErrorResponse
// @Router       /users/me/profile [patch]
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID := middleware.ContextUserID(r.Context())
	if err := h.svc.UpdateProfile(r.Context(), userID, req); err != nil {
		h.log.Error("user: update profile", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "profile updated"})
}

// GetSettings godoc
// @Summary      Get user settings
// @Description  Returns the authenticated user's app preferences (daily XP goal, SRS card count, reminders).
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} SettingsResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /users/me/settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	settings, err := h.svc.GetSettings(r.Context(), userID)
	if err != nil {
		h.log.Error("user: get settings", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, settings)
}

// UpdateSettings godoc
// @Summary      Update user settings
// @Description  Updates one or more app preferences for the authenticated user.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body UpdateSettingsRequest true "Settings fields to update (all optional)"
// @Success      200 {object} map[string]string
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      422 {object} response.ErrorResponse
// @Router       /users/me/settings [patch]
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID := middleware.ContextUserID(r.Context())
	if err := h.svc.UpdateSettings(r.Context(), userID, req); err != nil {
		h.log.Error("user: update settings", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "settings updated"})
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Changes the authenticated user's password after verifying the current one.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body ChangePasswordRequest true "Current and new password"
// @Success      200 {object} map[string]string
// @Failure      400 {object} response.ErrorResponse "Current password incorrect"
// @Failure      401 {object} response.ErrorResponse
// @Failure      422 {object} response.ErrorResponse
// @Router       /users/me/change-password [post]
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID := middleware.ContextUserID(r.Context())
	if err := h.svc.ChangePassword(r.Context(), userID, req); err != nil {
		response.BadRequest(w, "current password is incorrect")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}
