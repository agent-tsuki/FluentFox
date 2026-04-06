// Package xp — handler.go.
// HTTP handlers for XP and leaderboard endpoints. HTTP concerns only.
package xp

import (
	"net/http"

	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"go.uber.org/zap"
)

// Handler holds XP handler dependencies.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler constructs an XP Handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// GetXP handles GET /xp.
func (h *Handler) GetXP(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	xp, err := h.svc.GetXP(r.Context(), userID)
	if err != nil {
		h.log.Error("xp: get", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, xp)
}

// GetLeaderboard handles GET /xp/leaderboard.
func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.GetLeaderboard(r.Context())
	if err != nil {
		h.log.Error("xp: leaderboard", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, entries)
}
