// Package chapter — handler.go.
// HTTP handlers for chapter endpoints. HTTP concerns only.
package chapter

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/fluentfox/api/pkg/response"
	"go.uber.org/zap"
)

// Handler holds chapter handler dependencies.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler constructs a chapter Handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// List handles GET /chapters.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	chapters, total, err := h.svc.List(r.Context(), level, page, perPage)
	if err != nil {
		h.log.Error("chapter: list", zap.Error(err))
		response.InternalServerError(w)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, chapters, response.Meta{Total: total})
}

// GetDetail handles GET /chapters/{slug}.
func (h *Handler) GetDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	detail, err := h.svc.GetDetail(r.Context(), slug)
	if err != nil {
		response.NotFound(w)
		return
	}
	response.JSON(w, http.StatusOK, detail)
}
