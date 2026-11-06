package audio

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

const routeTag = "Audio"

// Fetching a stored clip needs no API call at all (it is on the CDN); only
// generating a new one needs a signed-in user, so the endpoint cannot be used
// anonymously as a free text-to-speech service.
func RegisterRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-audio-info",
		Method:      http.MethodGet,
		Path:        "/audio/info",
		Summary:     "Where the main voice's clips live on the CDN, and its credit line",
		Tags:        []string{routeTag},
		Metadata:    humautil.Public(),
	}, h.Info)

	huma.Register(api, huma.Operation{
		OperationID: "speak-texts",
		Method:      http.MethodPost,
		Path:        "/audio/speech",
		Summary:     "Get CDN URLs of spoken clips for texts, generating any that do not exist yet",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.Speak)
}
