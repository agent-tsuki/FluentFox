package learn

import (
	"context"

	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type Handler struct {
	settings *LearningSettingsService
	summary  *SummaryService
	logger   *zap.Logger
}

func NewHandler(settings *LearningSettingsService, summary *SummaryService, log *zap.Logger) *Handler {
	return &Handler{settings: settings, summary: summary, logger: log}
}

type EmptyInput struct{}

// GetSettings handles GET /learn/settings.
func (h *Handler) GetSettings(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[LearningSettingsResponse]], error) {
	resp, err := h.settings.Get(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// UpdateSettings handles PUT /learn/settings.
func (h *Handler) UpdateSettings(
	ctx context.Context, input *humautil.Input[UpdateLearningSettingsRequest],
) (*humautil.Output[humautil.APIResponse[LearningSettingsResponse]], error) {
	resp, err := h.settings.Update(ctx, middleware.ContextUserID(ctx), &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// GetSummary handles GET /learn/summary.
func (h *Handler) GetSummary(ctx context.Context, input *SummaryInput) (*humautil.Output[humautil.APIResponse[LearnSummary]], error) {
	resp, err := h.summary.Get(ctx, middleware.ContextUserID(ctx), input.Track)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
