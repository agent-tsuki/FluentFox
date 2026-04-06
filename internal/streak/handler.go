// Package streak — handler.go.
// HTTP handlers for streak endpoints. HTTP concerns only.
package streak

import (
	"net/http"

	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"go.uber.org/zap"
)

// Handler holds streak handler dependencies.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler constructs a streak Handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// GetStreak handles GET /streak.
func (h *Handler) GetStreak(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	streak, err := h.svc.GetStreak(r.Context(), userID)
	if err != nil {
		h.log.Error("streak: get", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, streak)
}
