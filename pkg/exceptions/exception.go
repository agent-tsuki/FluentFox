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

	ErrPasswordResetEmailSendFailed = New(http.StatusServiceUnavailable, "PASSWORD_RESET_EMAIL_SEND_FAILED", "we could not send the password reset email, please try again")
	ErrPasswordResetCodeInvalid     = New(http.StatusBadRequest, "PASSWORD_RESET_CODE_INVALID", "the password reset code is invalid or expired")
	ErrPasswordResetTokenInvalid    = New(http.StatusBadRequest, "PASSWORD_RESET_TOKEN_INVALID", "the password reset link is invalid or expired")

	ErrLoginCodeInvalid         = New(http.StatusBadRequest, "LOGIN_CODE_INVALID", "the sign-in code is invalid or expired")
	ErrLoginCodeCooldown        = New(http.StatusTooManyRequests, "LOGIN_CODE_COOLDOWN", "please wait before requesting another sign-in code")
	ErrLoginCodeDailyLimit      = New(http.StatusTooManyRequests, "LOGIN_CODE_DAILY_LIMIT", "sign-in code limit reached, please try again later")
	ErrLoginCodeEmailSendFailed = New(http.StatusServiceUnavailable, "LOGIN_CODE_EMAIL_SEND_FAILED", "we could not send the sign-in code, please try again")
)

var (
	ErrProfileNotFound            = New(http.StatusNotFound, "PROFILE_NOT_FOUND", "profile not found")
	ErrTimeZoneInvalid            = New(http.StatusUnprocessableEntity, "TIME_ZONE_INVALID", "time zone is not a valid IANA name, e.g. Asia/Tokyo")
	ErrAvatarPresetInvalid        = New(http.StatusUnprocessableEntity, "AVATAR_PRESET_INVALID", "unknown avatar preset")
	ErrAvatarTypeInvalid          = New(http.StatusUnsupportedMediaType, "AVATAR_TYPE_INVALID", "avatar must be a JPEG or PNG image")
	ErrAvatarImageInvalid         = New(http.StatusUnprocessableEntity, "AVATAR_IMAGE_INVALID", "image is corrupt, or too small or too large")
	ErrAvatarUploadUnavailable    = New(http.StatusServiceUnavailable, "AVATAR_UPLOAD_UNAVAILABLE", "avatar upload is not available right now")
	ErrSessionNotFound            = New(http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
	ErrCannotRevokeCurrentSession = New(http.StatusUnprocessableEntity, "CANNOT_REVOKE_CURRENT_SESSION", "use log out to end this session")
	ErrLocationInvalid            = New(http.StatusUnprocessableEntity, "LOCATION_INVALID", "latitude or longitude is out of range")
)

var (
	ErrKanaScriptNotFound = New(http.StatusNotFound, "KANA_SCRIPT_NOT_FOUND", "kana script not found")
	ErrKanjiLevelNotFound = New(http.StatusNotFound, "KANJI_LEVEL_NOT_FOUND", "kanji level not found")
	ErrKanjiNotFound      = New(http.StatusNotFound, "KANJI_NOT_FOUND", "kanji not found")
)

var (
	ErrLearningTrackInvalid    = New(http.StatusUnprocessableEntity, "LEARNING_TRACK_INVALID", "track must be hiragana, katakana or kanji_n5")
	ErrLearningSettingsInvalid = New(http.StatusUnprocessableEntity, "LEARNING_SETTINGS_INVALID", "a learning setting is out of range")
	ErrStudySessionNotFound    = New(http.StatusNotFound, "STUDY_SESSION_NOT_FOUND", "study session not found")
	ErrStudySessionCompleted   = New(http.StatusConflict, "STUDY_SESSION_COMPLETED", "this study session is already completed")
	ErrSRSCardNotFound         = New(http.StatusNotFound, "CARD_NOT_FOUND", "card not found")
	ErrQuestionTypeMismatch    = New(http.StatusUnprocessableEntity, "QUESTION_TYPE_MISMATCH", "question type does not fit the card")
)

var (
	ErrUnitNotFound            = New(http.StatusNotFound, "UNIT_NOT_FOUND", "unit not found")
	ErrUnitAmbiguous           = New(http.StatusUnprocessableEntity, "UNIT_AMBIGUOUS", "several tracks have this unit, pass track to choose one")
	ErrUnitProgressStepInvalid = New(http.StatusUnprocessableEntity, "UNIT_STEP_INVALID", "step must be watched, traced, written or practised")
	ErrPracticeModeInvalid     = New(http.StatusUnprocessableEntity, "PRACTICE_MODE_INVALID", "mode must be weak, unit, lookalike, speed or match")
	ErrPracticeUnitRequired    = New(http.StatusUnprocessableEntity, "PRACTICE_UNIT_REQUIRED", "unit practice needs a unit")
	ErrPracticePairInvalid     = New(http.StatusUnprocessableEntity, "PRACTICE_PAIR_INVALID", "pair must be two characters of the track, such as ぬ:め")
	ErrTestKindInvalid         = New(http.StatusUnprocessableEntity, "TEST_KIND_INVALID", "kind must be unit_quiz, skip or final")
	ErrTestScopeInvalid        = New(http.StatusUnprocessableEntity, "TEST_SCOPE_INVALID", "scope must be a unit slug, a group or all")
	ErrKanjiNoteTooLong        = New(http.StatusUnprocessableEntity, "KANJI_NOTE_TOO_LONG", "a note is at most 1000 characters")
)

var (
	ErrProgressDateInvalid   = New(http.StatusUnprocessableEntity, "PROGRESS_DATE_INVALID", "from and to must be YYYY-MM-DD dates at most 366 days apart")
	ErrProgressItemNotFound  = New(http.StatusNotFound, "PROGRESS_ITEM_NOT_FOUND", "item not found")
	ErrProgressItemAmbiguous = New(http.StatusUnprocessableEntity, "PROGRESS_ITEM_AMBIGUOUS", "several tracks have this item, pass track to choose one")
)
