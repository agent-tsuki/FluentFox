package exceptions

import (
	"errors"
	"net/http"
	"time"
)

// AppError is a domain error that carries an HTTP status code and a machine-readable code.
// Service layer returns these; the handler layer uses HandleError to write the response.
type AppError struct {
	Status  int    // HTTP status code to send (e.g. 409)
	Code    string // machine-readable code for clients (e.g. "EMAIL_ALREADY_IN_USE")
	Message string // human-readable message shown to the client
	Err     error  // optional wrapped cause — for internal logging only, never sent to client

	// Zero means no Retry-After header is sent.
	RetryAfter time.Duration
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

// Is makes errors.Is match copies made by WithCause and WithRetryAfter against
// the sentinel they came from.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	return ok && e.Status == t.Status && e.Code == t.Code && e.Message == t.Message
}

func WithCause(base *AppError, cause error) *AppError {
	copied := *base
	copied.Err = cause
	return &copied
}

// Sentinels are shared, so they must never be mutated in place.
func WithRetryAfter(base *AppError, wait time.Duration) *AppError {
	copied := *base
	copied.RetryAfter = wait
	return &copied
}

func EmailDomainTypo(suggested string) *AppError {
	return New(http.StatusUnprocessableEntity, "EMAIL_DOMAIN_TYPO", "did you mean "+suggested+"?")
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
	ErrAccountSuspended = New(http.StatusForbidden, "ACCOUNT_SUSPENDED", "your account is suspended, please contact support")
	ErrAccountDeleted   = New(http.StatusForbidden, "ACCOUNT_DELETED", "this account has been deleted, please contact support")
	ErrForbidden       = New(http.StatusForbidden, "FORBIDDEN", "you do not have permission to perform this action")

	// 404 Not Found
	ErrUserNotFound = New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	ErrUserEmailNotFound = New(http.StatusNotFound, "USER_NOT_FOUND", "user not found with email")

	// 400 Bad Request
	ErrBadRequest = New(http.StatusBadRequest, "BAD_REQUEST", "invalid request")
	ErrUpdateFail = New(http.StatusBadRequest, "BAD_REQUEST", "db update request failed")


	// 422 Unprocessable Entity
	ErrValidation = New(http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed")

	// 410 time expired
	ErrStatusGone = New(http.StatusGone, "REQUEST_EXPIRED", "request expired")
)

var (
	ErrDisposableEmail        = New(http.StatusUnprocessableEntity, "DISPOSABLE_EMAIL", "temporary email addresses are not allowed")
	ErrInvalidEmailDomain     = New(http.StatusUnprocessableEntity, "INVALID_EMAIL_DOMAIN", "this email domain cannot receive mail")
	ErrEmailUnchanged         = New(http.StatusUnprocessableEntity, "EMAIL_UNCHANGED", "new email is the same as the current one")
	ErrEmailAlreadyVerified   = New(http.StatusConflict, "EMAIL_ALREADY_VERIFIED", "email is already verified")
	ErrEmailNotVerified       = New(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "please verify your email to continue")
	ErrCurrentPasswordInvalid = New(http.StatusForbidden, "INVALID_CURRENT_PASSWORD", "current password is incorrect")
	ErrVerificationCooldown   = New(http.StatusTooManyRequests, "VERIFICATION_COOLDOWN", "please wait before requesting another verification email")
	ErrVerificationDailyLimit = New(http.StatusTooManyRequests, "VERIFICATION_DAILY_LIMIT", "verification email limit reached, please try again later")
	ErrEmailSendFailed        = New(http.StatusServiceUnavailable, "EMAIL_SEND_FAILED", "we could not send the verification email, please try again")


)
