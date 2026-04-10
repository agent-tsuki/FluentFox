// Package response provides standardised JSON envelope helpers.
// Every HTTP response in the application is written through this package.
// No handler may write raw JSON directly.
package response

import (
	"encoding/json"
	"net/http"

	"github.com/fluentfox/api/pkg/exceptions"
	"go.uber.org/zap"
)

// envelope is the outer wrapper for all API responses.
type envelope map[string]any

// Meta holds optional pagination or context metadata attached to success responses.
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
	Total      int `json:"total,omitempty"`
}

// errorBody is the shape of an error response.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the JSON envelope for all error responses.
// Used by Swagger documentation — mirrors the actual wire format.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the structured error detail inside ErrorResponse.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a 200 OK response with the given data wrapped in {"data": ...}.
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{"data": data})
}

// JSONWithMeta writes a success response with both data and meta fields.
func JSONWithMeta(w http.ResponseWriter, status int, data any, meta Meta) {
	write(w, status, envelope{"data": data, "meta": meta})
}

// Error writes a structured error response: {"error": {"code": ..., "message": ...}}.
func Error(w http.ResponseWriter, status int, code, message string) {
	write(w, status, envelope{"error": errorBody{Code: code, Message: message}})
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

// UnprocessableEntity writes a 422 response used for validation failures.
func UnprocessableEntity(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
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
	w.Header().Set("Retry-After", string(rune('0'+retryAfterSeconds)))
	Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests — please slow down")
}

// HandleError is the central error dispatcher for all HTTP handlers.
// If err is (or wraps) an *exceptions.AppError, it writes the correct status + JSON body.
// Any other error is treated as an internal server error: logged and returned as 500.
func HandleError(w http.ResponseWriter, err error, log *zap.Logger) {
	if appErr, ok := exceptions.As(err); ok {
		Error(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	log.Error("unhandled internal error", zap.Error(err))
	InternalServerError(w)
}

// write marshals the envelope to JSON and writes it to the response writer.
func write(w http.ResponseWriter, status int, payload envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(payload); err != nil {
		// At this point headers are already written; we cannot change the status.
		// Log this at the call site if needed.
		_ = err
	}
}
