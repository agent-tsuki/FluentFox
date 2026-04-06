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

// GetMe handles GET /users/me.
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

// GetProfile handles GET /users/me/profile.
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

// UpdateProfile handles PATCH /users/me/profile.
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

// GetSettings handles GET /users/me/settings.
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

// UpdateSettings handles PATCH /users/me/settings.
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

// ChangePassword handles POST /users/me/change-password.
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
