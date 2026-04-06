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

// List godoc
// @Summary      List chapters
// @Description  Returns a paginated list of published chapters, optionally filtered by JLPT level.
// @Tags         chapters
// @Produce      json
// @Security     BearerAuth
// @Param        level    query string false "JLPT level filter" Enums(N5,N4,N3,N2,N1)
// @Param        page     query int    false "Page number (default: 1)"
// @Param        per_page query int    false "Results per page (default: 20)"
// @Success      200 {array} ChapterResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /chapters [get]
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

// GetDetail godoc
// @Summary      Get chapter detail
// @Description  Returns a single chapter with its grammar concepts and cultural insights.
// @Tags         chapters
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Chapter slug"
// @Success      200 {object} ChapterDetailResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      404 {object} response.ErrorResponse
// @Router       /chapters/{slug} [get]
func (h *Handler) GetDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	detail, err := h.svc.GetDetail(r.Context(), slug)
	if err != nil {
		response.NotFound(w)
		return
	}
	response.JSON(w, http.StatusOK, detail)
}
