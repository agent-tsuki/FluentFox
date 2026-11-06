package audio

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

const routeTag = "Audio"

// Anyone may look up stored clips (/audio/clips); generating a new one
// (/audio/speech) needs a signed-in user, so the API cannot be used
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
		OperationID: "find-audio-clips",
		Method:      http.MethodPost,
		Path:        "/audio/clips",
		Summary:     "Get CDN URLs of the spoken clips that already exist for texts (never generates)",
		Tags:        []string{routeTag},
		Metadata:    humautil.Public(),
	}, h.Find)

	huma.Register(api, huma.Operation{
		OperationID: "speak-texts",
		Method:      http.MethodPost,
		Path:        "/audio/speech",
		Summary:     "Get CDN URLs of spoken clips for texts, generating any that do not exist yet",
		Tags:        []string{routeTag},
		Security:    humautil.BearerSecurity(),
	}, h.Speak)
}
