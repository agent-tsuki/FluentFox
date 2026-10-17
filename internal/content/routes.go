package content

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/humautil"
)

// Guests may browse kana and N5 content, so every content route is public.
func RegisterRoutes(api huma.API, h *Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-kana-course",
		Method:      http.MethodGet,
		Path:        "/content/kana/{script}",
		Summary:     "Get the units, groups and kana of a kana script",
		Tags:        []string{"Content"},
		Metadata:    humautil.Public(),
	}, h.GetKanaCourse)

	huma.Register(api, huma.Operation{
		OperationID: "get-kanji-level-index",
		Method:      http.MethodGet,
		Path:        "/content/kanji/{level}/index",
		Summary:     "Get the light index of a kanji level (units, groups, rows)",
		Tags:        []string{"Content"},
		Metadata:    humautil.Public(),
	}, h.GetKanjiLevelIndex)

	huma.Register(api, huma.Operation{
		OperationID: "get-kanji-level-detail",
		Method:      http.MethodGet,
		Path:        "/content/kanji/{level}/detail",
		Summary:     "Get every full kanji entry of a level",
		Tags:        []string{"Content"},
		Metadata:    humautil.Public(),
	}, h.GetKanjiLevelDetail)

	huma.Register(api, huma.Operation{
		OperationID: "get-kanji-entry",
		Method:      http.MethodGet,
		Path:        "/content/kanji/chars/{char}",
		Summary:     "Get one kanji entry",
		Tags:        []string{"Content"},
		Metadata:    humautil.Public(),
	}, h.GetKanjiEntry)
}
