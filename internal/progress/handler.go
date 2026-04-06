// Package progress — handler.go.
// HTTP handlers for progress endpoints.
package progress

import (
	"net/http"

	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"go.uber.org/zap"
)

// Handler holds progress handler dependencies.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler constructs a progress Handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// GetOverall godoc
// @Summary      Get overall progress
// @Description  Returns a summary of the user's progress across all chapters and vocabulary.
// @Tags         progress
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} OverallProgressResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /progress [get]
func (h *Handler) GetOverall(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	prog, err := h.svc.GetOverallProgress(r.Context(), userID)
	if err != nil {
		h.log.Error("progress: get overall", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, prog)
}
