package content

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type ChapterHandler struct {
	chapters *ChapterContentService
	logger   *zap.Logger
}

func NewChapterHandler(chapters *ChapterContentService, log *zap.Logger) *ChapterHandler {
	return &ChapterHandler{chapters: chapters, logger: log}
}

const contentTag = "Content"

// Public like the other content: guests may read a chapter's words.
func RegisterChapterRoutes(api huma.API, h *ChapterHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "get-grammar-chapter-content",
		Method:      http.MethodGet,
		Path:        "/content/grammar/{level}/chapters/{chapter}",
		Summary:     "Get a grammar chapter's words and its practice sentences read in full",
		Tags:        []string{contentTag},
		Metadata:    humautil.Public(),
	}, h.GetChapter)
}

// GetChapter handles GET /content/grammar/{level}/chapters/{chapter}.
func (h *ChapterHandler) GetChapter(ctx context.Context, input *ChapterContentInput) (*humautil.Output[humautil.APIResponse[ChapterContentResponse]], error) {
	resp, err := h.chapters.Get(ctx, input.Level, input.Chapter)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
