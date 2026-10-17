package content

import (
	"context"

	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type Handler struct {
	kana   *KanaContentService
	kanji  *KanjiContentService
	logger *zap.Logger
}

func NewHandler(kana *KanaContentService, kanji *KanjiContentService, log *zap.Logger) *Handler {
	return &Handler{kana: kana, kanji: kanji, logger: log}
}

// GetKanaCourse handles GET /content/kana/{script}.
func (h *Handler) GetKanaCourse(ctx context.Context, input *KanaScriptInput) (*humautil.Output[humautil.APIResponse[KanaCourseResponse]], error) {
	resp, err := h.kana.GetCourse(ctx, input.Script)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetKanjiLevelIndex handles GET /content/kanji/{level}/index.
func (h *Handler) GetKanjiLevelIndex(ctx context.Context, input *KanjiLevelInput) (*humautil.Output[humautil.APIResponse[KanjiLevelIndex]], error) {
	resp, err := h.kanji.GetLevelIndex(ctx, input.Level)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetKanjiLevelDetail handles GET /content/kanji/{level}/detail.
func (h *Handler) GetKanjiLevelDetail(ctx context.Context, input *KanjiLevelInput) (*humautil.Output[humautil.APIResponse[KanjiLevelDetail]], error) {
	resp, err := h.kanji.GetLevelDetail(ctx, input.Level)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetKanjiEntry handles GET /content/kanji/chars/{char}.
func (h *Handler) GetKanjiEntry(ctx context.Context, input *KanjiCharInput) (*humautil.Output[humautil.APIResponse[KanjiEntryResponse]], error) {
	resp, err := h.kanji.GetEntry(ctx, input.Char)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
