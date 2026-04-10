package exceptions

import (
	"errors"
	"net/http"
)

// AppError is a domain error that carries an HTTP status code and a machine-readable code.
// Service layer returns these; the handler layer uses HandleError to write the response.
type AppError struct {
	Status  int    // HTTP status code to send (e.g. 409)
	Code    string // machine-readable code for clients (e.g. "EMAIL_ALREADY_IN_USE")
	Message string // human-readable message shown to the client
	Err     error  // optional wrapped cause — for internal logging only, never sent to client
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// New creates an AppError without a wrapped cause.
func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// Wrap creates an AppError that preserves an underlying error for logging.
func Wrap(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: cause}
}

// As returns the AppError if err (or any error in its chain) is an *AppError.
func As(err error) (*AppError, bool) {
	var e *AppError
	return e, errors.As(err, &e)
}

// ── Sentinel errors — return these from the service layer ────────────────────

var (
	// 409 Conflict
	ErrEmailAlreadyInUse    = New(http.StatusConflict, "EMAIL_ALREADY_IN_USE", "email is already registered")
	ErrUsernameAlreadyInUse = New(http.StatusConflict, "USERNAME_ALREADY_IN_USE", "username is already taken")

	// 401 Unauthorized
	ErrInvalidCredentials = New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
	ErrTokenExpired       = New(http.StatusUnauthorized, "TOKEN_EXPIRED", "token has expired")
	ErrTokenInvalid       = New(http.StatusUnauthorized, "TOKEN_INVALID", "token is invalid")

	// 403 Forbidden
	ErrAccountInactive = New(http.StatusForbidden, "ACCOUNT_INACTIVE", "account is inactive")
	ErrForbidden       = New(http.StatusForbidden, "FORBIDDEN", "you do not have permission to perform this action")

	// 404 Not Found
	ErrUserNotFound = New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")

	// 400 Bad Request
	ErrBadRequest = New(http.StatusBadRequest, "BAD_REQUEST", "invalid request")

	// 422 Unprocessable Entity
	ErrValidation = New(http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed")
)
