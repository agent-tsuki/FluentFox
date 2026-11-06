package audio

import (
	"context"

	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type Handler struct {
	speech *SpeechService
	logger *zap.Logger
}

func NewHandler(speech *SpeechService, log *zap.Logger) *Handler {
	return &Handler{speech: speech, logger: log}
}

type EmptyInput struct{}

func (h *Handler) Info(_ context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[SpeechInfo]], error) {
	return humautil.OK(h.speech.Info()), nil
}

func (h *Handler) Speak(ctx context.Context, input *humautil.Input[SpeechRequest]) (*humautil.Output[humautil.APIResponse[SpeechResponse]], error) {
	who := Requester{UserID: middleware.ContextUserID(ctx), Admin: middleware.ContextIsAdmin(ctx)}
	resp, err := h.speech.Speak(ctx, who, input.Body.Texts)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
