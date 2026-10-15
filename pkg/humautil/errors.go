package humautil

import (
	"math"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// APIError is the canonical JSON error envelope returned by all huma endpoints.
//
//	{ "status": "failed", "error": { "log_id": "...", "error_code": "...", "message": "..." } }
type APIError struct {
	httpStatus int
	headers    http.Header
	Status     string    `json:"status"`
	Err        ErrDetail `json:"error"`
}

// ErrDetail holds the structured error payload.
type ErrDetail struct {
	LogID     string `json:"log_id"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func (e *APIError) GetStatus() int { return e.httpStatus }
func (e *APIError) Error() string  { return e.Err.Message }

// Implements huma.HeadersError so Retry-After reaches the response.
func (e *APIError) GetHeaders() http.Header { return e.headers }

func newAPIError(status int, code, message string) *APIError {
	return &APIError{
		httpStatus: status,
		Status:     "failed",
		Err: ErrDetail{
			LogID:     uuid.New().String(),
			ErrorCode: code,
			Message:   message,
		},
	}
}

// InitHumaErrors overrides huma's default error formatter with our canonical shape.
// Call this once after creating the huma.API instance in main.
func InitHumaErrors() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		return newAPIError(status, statusToCode(status), msg)
	}
}

// MapErr converts a domain error into a huma HTTP error using our canonical shape.
// AppErrors carry their own status and code; anything else becomes a 500.
func MapErr(err error, log *zap.Logger) error {
	if appErr, ok := exceptions.As(err); ok {
		apiErr := newAPIError(appErr.Status, appErr.Code, appErr.Message)
		if appErr.RetryAfter > 0 {
			// Round up so a client never retries a fraction of a second too early.
			seconds := int(math.Ceil(appErr.RetryAfter.Seconds()))
			apiErr.headers = http.Header{"Retry-After": []string{strconv.Itoa(seconds)}}
		}
		return apiErr
	}
	log.Error("unhandled internal error", zap.Error(err))
	return newAPIError(http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}

// statusToCode maps an HTTP status to a machine-readable error code used when
// huma itself raises an error (e.g. request body validation, unknown route).
func statusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusGone:
		return "GONE"
	case http.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	case http.StatusTooManyRequests:
		return "RATE_LIMITED"
	default:
		return "INTERNAL_ERROR"
	}
}
