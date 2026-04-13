package auth

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterRoutes mounts auth endpoints onto the huma API.
// OpenAPI metadata (summary, tags, responses) is declared here so the
// auto-generated /openapi.json stays accurate without touching handler logic.
func RegisterRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID:   "register-user",
		Method:        http.MethodPost,
		Path:          "/auth/register",
		Summary:       "Register a new user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusCreated,
	}, h.AuthRegister)

	huma.Register(api, huma.Operation{
		OperationID: "verify-email",
		Method:      http.MethodPost,
		Path:        "/auth/verify",
		Summary:     "Verify user email address",
		Tags:        []string{"Auth"},
	}, h.AuthVerify)

	huma.Register(api, huma.Operation{
		OperationID:   "login-user",
		Method:        http.MethodPost,
		Path:          "/auth/login",
		Summary:       "Login user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, h.Login)

	huma.Register(api, huma.Operation{
		OperationID:   "refresh-token",
		Method:        http.MethodPost,
		Path:          "/auth/refresh",
		Summary:       "Refresh access token",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, h.Refresh)

	huma.Register(api, huma.Operation{
		OperationID:   "logout-user",
		Method:        http.MethodPost,
		Path:          "/auth/logout",
		Summary:       "Logout user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, h.Logout)
}
