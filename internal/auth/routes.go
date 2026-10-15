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

	// Refresh and logout identify the user by the refresh token in the body,
	// so they work even when the access token has expired.
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
