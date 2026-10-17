package learn

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

// No route here is Public(), so every one needs a token and a verified email.
func RegisterRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-learning-settings",
		Method:      http.MethodGet,
		Path:        "/learn/settings",
		Summary:     "Get the learning settings, created with defaults on first read",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.GetSettings)

	huma.Register(api, huma.Operation{
		OperationID: "update-learning-settings",
		Method:      http.MethodPut,
		Path:        "/learn/settings",
		Summary:     "Change some learning settings",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.UpdateSettings)

	huma.Register(api, huma.Operation{
		OperationID: "get-learn-summary",
		Method:      http.MethodGet,
		Path:        "/learn/summary",
		Summary:     "Get due counts, new items left today, next unit and goal progress",
		Tags:        []string{"Learn"},
		Security:    humautil.BearerSecurity(),
	}, h.GetSummary)
}
