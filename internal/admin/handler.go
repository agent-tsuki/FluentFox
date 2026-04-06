// Package admin — handler.go.
// HTTP handlers for admin endpoints. All routes require admin role.
package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler holds admin handler dependencies.
type Handler struct {
	svc       *Service
	validator *validator.Validator
	log       *zap.Logger
}

// NewHandler constructs an admin Handler.
func NewHandler(svc *Service, v *validator.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: v, log: log}
}

// GetStats godoc
// @Summary      Get platform stats
// @Description  Returns platform-wide statistics (total users, active today, reviews, published chapters). Requires admin role.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} StatsResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /admin/stats [get]
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		h.log.Error("admin: get stats", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}

// ListUsers godoc
// @Summary      List all users
// @Description  Returns a paginated list of all users with admin-only fields. Requires admin role.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page     query int false "Page number"
// @Param        per_page query int false "Results per page"
// @Success      200 {array} AdminUserResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /admin/users [get]
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	users, total, err := h.svc.ListUsers(r.Context(), page, perPage)
	if err != nil {
		h.log.Error("admin: list users", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, users, response.Meta{Total: total})
}

// BanUser godoc
// @Summary      Ban a user
// @Description  Bans a user account with a mandatory reason. Requires admin role.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path string        true "Target user UUID"
// @Param        body body BanUserRequest true "Ban reason"
// @Success      200 {object} map[string]string
// @Failure      400 {object} response.ErrorResponse "Invalid user ID"
// @Failure      401 {object} response.ErrorResponse
// @Failure      403 {object} response.ErrorResponse
// @Failure      422 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /admin/users/{id}/ban [post]
func (h *Handler) BanUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid user id")
		return
	}

	var req BanUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	adminID := middleware.ContextUserID(r.Context())
	if err := h.svc.BanUser(r.Context(), adminID, targetID, req.Reason); err != nil {
		h.log.Error("admin: ban user", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "user banned"})
}
