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
}
