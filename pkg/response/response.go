// Package response provides standardised JSON envelope helpers.
// Every HTTP response in the application is written through this package.
// No handler may write raw JSON directly.
package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// envelope is the outer wrapper for success API responses.
type envelope map[string]any

// Meta holds optional pagination or context metadata attached to success responses.
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
	Total      int `json:"total,omitempty"`
}

// errDetail is the structured error payload inside every error response.
type errDetail struct {
	LogID     string                 `json:"log_id"`
	ErrorCode string                 `json:"error_code"`
	Message   string                 `json:"message"`
	Fields    []validator.FieldError `json:"fields,omitempty"`
}

// errEnvelope is the outer wrapper for all error responses.
//
//	{ "status": "failed", "error": { "log_id": "...", "error_code": "...", "message": "..." } }
type errEnvelope struct {
	Status string    `json:"status"`
	Error  errDetail `json:"error"`
}

// ErrorResponse is the JSON envelope for all error responses (used by OpenAPI docs).
type ErrorResponse struct {
	Status string    `json:"status"`
	Error  ErrorBody `json:"error"`
}

// ErrorBody is the structured error detail inside ErrorResponse.
type ErrorBody struct {
	LogID     string `json:"log_id"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// JSON writes a success response with the given data wrapped in {"data": ...}.
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{"data": data})
}

// JSONWithMeta writes a success response with both data and meta fields.
func JSONWithMeta(w http.ResponseWriter, status int, data any, meta Meta) {
	write(w, status, envelope{"data": data, "meta": meta})
}

// Error writes a structured error response:
//
//	{ "status": "failed", "error": { "log_id": "...", "error_code": "...", "message": "..." } }
func Error(w http.ResponseWriter, status int, code, message string) {
	writeError(w, status, errDetail{
		LogID:     uuid.New().String(),
		ErrorCode: code,
		Message:   message,
	})
}

// NotFound writes a 404 response.
func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "NOT_FOUND", "the requested resource was not found")
}

// Unauthorized writes a 401 response.
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "authentication required"
	}
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden writes a 403 response.
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, "FORBIDDEN", "you do not have permission to perform this action")
}

// BadRequest writes a 400 response.
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// UnprocessableEntity writes a 422 response used for simple validation failures.
func UnprocessableEntity(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
}

// ValidationErrors writes a 422 response with per-field error details.
func ValidationErrors(w http.ResponseWriter, fields []validator.FieldError) {
	writeError(w, http.StatusUnprocessableEntity, errDetail{
		LogID:     uuid.New().String(),
		ErrorCode: "VALIDATION_ERROR",
		Message:   "validation failed",
		Fields:    fields,
	})
}

// InternalServerError writes a 500 response without leaking internal details.
func InternalServerError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}

// Conflict writes a 409 response.
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, "CONFLICT", message)
}

// TooManyRequests writes a 429 response with a Retry-After header.
func TooManyRequests(w http.ResponseWriter, retryAfterSeconds int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests — please slow down")
}

// HandleError is the central error dispatcher for all HTTP handlers.
// Dispatch order:
//  1. *validator.ValidationError  → 422 with per-field details
//  2. *exceptions.AppError        → status + code from the error
//  3. anything else               → 500, logged internally
func HandleError(w http.ResponseWriter, err error, log *zap.Logger) {
	var valErr *validator.ValidationError
	if errors.As(err, &valErr) {
		ValidationErrors(w, valErr.Fields)
		return
	}

	if appErr, ok := exceptions.As(err); ok {
		Error(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	log.Error("unhandled internal error", zap.Error(err))
	InternalServerError(w)
}

// writeError marshals an errEnvelope to the response writer.
func writeError(w http.ResponseWriter, status int, detail errDetail) {
	write(w, status, errEnvelope{Status: "failed", Error: detail})
}

// write marshals the payload to JSON and writes it to the response writer.
func write(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(payload); err != nil {
		_ = err
	}
}
