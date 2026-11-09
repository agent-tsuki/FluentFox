package learn

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type ChapterHandler struct {
	completion *ChapterCompletionService
	logger     *zap.Logger
}

func NewChapterHandler(completion *ChapterCompletionService, log *zap.Logger) *ChapterHandler {
	return &ChapterHandler{completion: completion, logger: log}
}

const learnTag = "Learn"

// Needs a token and a verified email, like the other learn routes.
func RegisterChapterRoutes(api huma.API, h *ChapterHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "set-grammar-chapter-completion",
		Method:      http.MethodPut,
		Path:        "/learn/grammar/{level}/chapters/{chapter}/completion",
		Summary:     "Mark a grammar chapter complete (its words and sentences join reviews) or take the mark away",
		Tags:        []string{learnTag},
		Security:    humautil.BearerSecurity(),
	}, h.SetCompletion)
}

// SetCompletion handles PUT /learn/grammar/{level}/chapters/{chapter}/completion.
func (h *ChapterHandler) SetCompletion(ctx context.Context, input *ChapterCompletionInput) (*humautil.Output[humautil.APIResponse[UnitProgressData]], error) {
	resp, err := h.completion.Set(ctx, middleware.ContextUserID(ctx), input)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
