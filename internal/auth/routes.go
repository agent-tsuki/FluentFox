package auth

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

// RegisterRoutes mounts auth endpoints onto the huma API.
// OpenAPI metadata (summary, tags, responses) is declared here so the
// auto-generated /openapi.json stays accurate without touching handler logic.
// Every route needs a valid access token unless it is marked humautil.Public().
func RegisterRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID:   "request-password-reset-code",
		Method:        http.MethodPost,
		Path:          "/auth/forgot-password",
		Summary:       "Send a password-reset verification code",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Metadata:      humautil.Public(),
	}, h.ForgotPassword)

	huma.Register(api, huma.Operation{
		OperationID:   "verify-password-reset-code",
		Method:        http.MethodPost,
		Path:          "/auth/forgot-password/verify-code",
		Summary:       "Verify a password-reset code and send a reset link",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Metadata:      humautil.Public(),
	}, h.VerifyPasswordResetCode)

	huma.Register(api, huma.Operation{
		OperationID:   "reset-password",
		Method:        http.MethodPost,
		Path:          "/auth/reset-password",
		Summary:       "Reset a password using a one-time email link token",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Metadata:      humautil.Public(),
	}, h.ResetPassword)

	huma.Register(api, huma.Operation{
		OperationID:   "register-user",
		Method:        http.MethodPost,
		Path:          "/auth/register",
		Summary:       "Register a new user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusCreated,
		Metadata:      humautil.Public(),
	}, h.AuthRegister)

	huma.Register(api, huma.Operation{
		OperationID: "verify-email",
		Method:      http.MethodPost,
		Path:        "/auth/verify",
		Summary:     "Verify user email address",
		Tags:        []string{"Auth"},
		Metadata:    humautil.Public(),
	}, h.AuthVerify)

	huma.Register(api, huma.Operation{
		OperationID:   "login-user",
		Method:        http.MethodPost,
		Path:          "/auth/login",
		Summary:       "Login user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Metadata:      humautil.Public(),
	}, h.Login)

	huma.Register(api, huma.Operation{
		OperationID:   "verify-login-code",
		Method:        http.MethodPost,
		Path:          "/auth/login/verify-code",
		Summary:       "Finish a login with the emailed code (sent when the login came from a VPN)",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Metadata:      humautil.Public(),
	}, h.VerifyLoginCode)

	// Refresh and logout identify the user from the HttpOnly refresh cookie, so
	// they work even when the in-memory access token has expired.
	huma.Register(api, huma.Operation{
		OperationID:   "refresh-token",
		Method:        http.MethodPost,
		Path:          "/auth/refresh",
		Summary:       "Refresh access token",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Metadata:      humautil.Public(),
	}, h.Refresh)

	huma.Register(api, huma.Operation{
		OperationID:   "logout-user",
		Method:        http.MethodPost,
		Path:          "/auth/logout",
		Summary:       "Logout user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Metadata:      humautil.Public(),
	}, h.Logout)

	huma.Register(api, huma.Operation{
		OperationID:   "resend-verification",
		Method:        http.MethodPost,
		Path:          "/auth/verification/resend",
		Summary:       "Resend the email verification link to the logged-in user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Security:      humautil.BearerSecurity(),
		Metadata:      humautil.AllowUnverified(),
	}, h.ResendVerification)

	huma.Register(api, huma.Operation{
		OperationID:   "change-email",
		Method:        http.MethodPost,
		Path:          "/auth/change-email",
		Summary:       "Change the email of an unverified account",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
		Security:      humautil.BearerSecurity(),
		Metadata:      humautil.AllowUnverified(),
	}, h.ChangeEmail)
}
