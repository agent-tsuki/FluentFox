// Package srs — handler.go.
// HTTP handlers for SRS endpoints. HTTP concerns only — no SQL, no business logic.
package srs

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/validator"
	"go.uber.org/zap"
)

// Handler holds SRS handler dependencies.
type Handler struct {
	svc       *Service
	validator *validator.Validator
	log       *zap.Logger
}

// NewHandler constructs an SRS Handler.
func NewHandler(svc *Service, v *validator.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: v, log: log}
}

// GetDueCards handles GET /srs/due.
func (h *Handler) GetDueCards(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	cards, err := h.svc.GetDueCards(r.Context(), userID, limit)
	if err != nil {
		h.log.Error("srs: get due cards", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, cards)
}

// GetDueCount handles GET /srs/due/count.
func (h *Handler) GetDueCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())

	counts, err := h.svc.GetDueCount(r.Context(), userID)
	if err != nil {
		h.log.Error("srs: get due count", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, counts)
}

// SubmitReview handles POST /srs/review.
func (h *Handler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	var req SubmitReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID := middleware.ContextUserID(r.Context())
	result, err := h.svc.SubmitReview(r.Context(), userID, req)
	if err != nil {
		h.log.Error("srs: submit review", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
