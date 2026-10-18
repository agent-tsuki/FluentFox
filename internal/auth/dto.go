package auth

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/fluentfox/api/pkg/humautil"
)

type RegisterRequest struct {
	Email       string  `json:"email"        validate:"required,email"   format:"email"   doc:"User's email address"`
	UserName    *string `json:"user_name,omitempty"                                          doc:"Username; auto-generated if omitted"`
	FirstName   string  `json:"first_name"   validate:"required"                             doc:"First name"`
	LastName    *string `json:"last_name,omitempty"                                          doc:"Last name"`
	PhoneNumber *string `json:"phone_number,omitempty"                                       doc:"Phone number"`
	NativeLang  string  `json:"native_lang"  validate:"required"                             doc:"Native language code (e.g. en, ja)"`
	Password    string  `json:"password"     validate:"required,min=8,max=128"   minLength:"8" maxLength:"128" doc:"Password — 8 to 128 characters"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email" format:"email" doc:"User's email address"`
	Password string `json:"password" validate:"required,min=8,max=128" minLength:"8" maxLength:"128" doc:"Password — 8 to 128 characters"`
}

// UserSummary carries the fields a client needs immediately after auth.
type UserSummary struct {
	UserID        uuid.UUID `json:"user_id"        doc:"User's unique identifier"`
	Username      string    `json:"username"       doc:"User's handle"`
	IsAdmin       bool      `json:"is_admin"       doc:"User role: user or admin"`
	EmailVerified bool      `json:"email_verified" doc:"User verified account or not"`
	IsActive      bool      `json:"is_active"      doc:"Account is active or suspended"`
}

type LoginResponse struct {
	// omitzero: a login that needs an emailed code has no token and no user yet.
	AccessToken string `json:"access_token,omitzero" doc:"Short-lived JWT access token"`
	// RefreshToken is internal-only. It is written as an HttpOnly response
	// cookie by the HTTP handler and must never appear in JSON or OpenAPI.
	RefreshToken string      `json:"-"`
	User         UserSummary `json:"user,omitzero"   doc:"Authenticated user summary"`
	// Set instead of tokens when the login needs an emailed code first.
	LoginChallenge *LoginChallengeResponse `json:"login_challenge,omitempty" doc:"Present when a login needs an emailed code; no tokens are issued"`
}

type LoginChallengeResponse struct {
	ChallengeID      uuid.UUID `json:"challenge_id"       doc:"Send back with the code to POST /auth/login/verify-code"`
	Reason           string    `json:"reason"             enum:"vpn" doc:"Why a code is needed"`
	EmailHint        string    `json:"email_hint"         doc:"Masked address the code was sent to, e.g. s****@gmail.com"`
	ExpiresInSeconds int       `json:"expires_in_seconds" doc:"Seconds until the code expires"`
}

type VerifyLoginCodeRequest struct {
	ChallengeID uuid.UUID `json:"challenge_id" validate:"required" doc:"challenge_id from the login response"`
	Code        string    `json:"code"         validate:"required,len=6,numeric" minLength:"6" maxLength:"6" doc:"6-digit code from the email"`
}

// Embedding keeps the session fields flat in JSON, the same shape as /auth/login.
type RegisterResponse struct {
	Message string `json:"message" doc:"Human-readable response message"`
	LoginResponse
}

// RefreshCookieInput reads the opaque refresh credential from the HttpOnly
// cookie rather than a JSON body, keeping it inaccessible to JavaScript.
type RefreshCookieInput struct {
	RefreshToken string `cookie:"__Host-ff_refresh" doc:"HttpOnly refresh-session cookie"`
}

// SessionOutput is the only response shape that can set the refresh cookie.
// Cache-Control prevents browsers and intermediaries from storing access-token
// responses. Huma serialises each http.Cookie into a Set-Cookie header.
type SessionOutput[T any] struct {
	SetCookie    []http.Cookie `header:"Set-Cookie"`
	CacheControl string        `header:"Cache-Control"`
	Body         humautil.APIResponse[T]
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email" format:"email" doc:"Account email address"`
}

type VerifyPasswordResetCodeRequest struct {
	Email string `json:"email" validate:"required,email" format:"email" doc:"Account email address"`
	Code  string `json:"code" validate:"required,len=8,alphanum" minLength:"8" maxLength:"8" doc:"Uppercase password-reset code sent by email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required" doc:"One-time reset token from the email link"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128" minLength:"8" maxLength:"128" doc:"New password — 8 to 128 characters"`
}

type ChangeEmailRequest struct {
	NewEmail        string `json:"new_email"        format:"email" doc:"Corrected email address"`
	CurrentPassword string `json:"current_password" minLength:"8" maxLength:"128" doc:"Current account password"`
}

type EmptyInput struct{}
