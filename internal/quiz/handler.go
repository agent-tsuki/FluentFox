// Package quiz — handler.go.
// HTTP handlers for quiz endpoints. HTTP concerns only.
package quiz

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler holds quiz handler dependencies.
type Handler struct {
	svc       *Service
	validator *validator.Validator
	log       *zap.Logger
}

// NewHandler constructs a quiz Handler.
func NewHandler(svc *Service, v *validator.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: v, log: log}
}

// StartSession handles POST /quiz/sessions.
func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID := middleware.ContextUserID(r.Context())
	session, err := h.svc.StartSession(r.Context(), userID, req)
	if err != nil {
		h.log.Error("quiz: start session", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusCreated, session)
}

// SubmitAnswer handles POST /quiz/sessions/{id}/answers.
func (h *Handler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid session id")
		return
	}

	var req SubmitAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID := middleware.ContextUserID(r.Context())
	result, err := h.svc.SubmitAnswer(r.Context(), userID, sessionID, req)
	if err != nil {
		h.log.Error("quiz: submit answer", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// FinishSession handles POST /quiz/sessions/{id}/finish.
func (h *Handler) FinishSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid session id")
		return
	}

	userID := middleware.ContextUserID(r.Context())
	summary, err := h.svc.FinishSession(r.Context(), userID, sessionID)
	if err != nil {
		h.log.Error("quiz: finish session", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}
