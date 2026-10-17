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
	lessons  *StartLessonService
	reviews  *StartReviewService
	answers  *SubmitAnswersService
	complete *CompleteSessionService
	logger   *zap.Logger
}

func NewHandler(
	settings *LearningSettingsService,
	summary *SummaryService,
	lessons *StartLessonService,
	reviews *StartReviewService,
	answers *SubmitAnswersService,
	complete *CompleteSessionService,
	log *zap.Logger,
) *Handler {
	return &Handler{
		settings: settings, summary: summary, lessons: lessons, reviews: reviews,
		answers: answers, complete: complete, logger: log,
	}
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

// StartLesson handles POST /learn/lessons.
func (h *Handler) StartLesson(ctx context.Context, input *humautil.Input[StartLessonRequest]) (*humautil.Output[humautil.APIResponse[SessionStart]], error) {
	resp, err := h.lessons.Start(ctx, middleware.ContextUserID(ctx), &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// StartReview handles POST /learn/reviews.
func (h *Handler) StartReview(ctx context.Context, input *humautil.Input[StartReviewRequest]) (*humautil.Output[humautil.APIResponse[SessionStart]], error) {
	resp, err := h.reviews.Start(ctx, middleware.ContextUserID(ctx), &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// SubmitAnswers handles POST /learn/sessions/{id}/answers.
func (h *Handler) SubmitAnswers(ctx context.Context, input *SubmitAnswersInput) (*humautil.Output[humautil.APIResponse[SubmitAnswersResponse]], error) {
	resp, err := h.answers.Submit(ctx, middleware.ContextUserID(ctx), input.ID, &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// CompleteSession handles POST /learn/sessions/{id}/complete.
func (h *Handler) CompleteSession(ctx context.Context, input *SessionIDInput) (*humautil.Output[humautil.APIResponse[SessionResult]], error) {
	resp, err := h.complete.Complete(ctx, middleware.ContextUserID(ctx), input.ID)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
