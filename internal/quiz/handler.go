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

// StartSession godoc
// @Summary      Start a quiz session
// @Description  Creates a new quiz session of the given type. Optionally scoped to a specific chapter.
// @Tags         quiz
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body StartSessionRequest true "Quiz type, optional chapter, card count"
// @Success      201 {object} SessionResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      422 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /quiz/sessions [post]
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

// SubmitAnswer godoc
// @Summary      Submit a quiz answer
// @Description  Submits an answer for one item in an active quiz session. Returns correctness and XP earned.
// @Tags         quiz
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path string            true  "Quiz session UUID"
// @Param        body body SubmitAnswerRequest true "Answer payload"
// @Success      200 {object} AnswerResultResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      422 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /quiz/sessions/{id}/answers [post]
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

// FinishSession godoc
// @Summary      Finish a quiz session
// @Description  Marks the quiz session as completed and returns a summary with accuracy and total XP earned.
// @Tags         quiz
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Quiz session UUID"
// @Success      200 {object} SessionSummaryResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /quiz/sessions/{id}/finish [post]
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
