package audio

import (
	"context"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type Handler struct {
	speech *SpeechService
	// alt speaks the other side of a dialogue; nil when there is no second voice.
	alt    *SpeechService
	logger *zap.Logger
}

func NewHandler(speech *SpeechService, log *zap.Logger) *Handler {
	return &Handler{speech: speech, logger: log}
}

// WithAltVoice adds the second voice (voice: alt). Its clips live in their
// own folder, and it has its own hourly limit.
func (h *Handler) WithAltVoice(alt *SpeechService) *Handler {
	if alt != nil && alt.enabled() {
		h.alt = alt
	}
	return h
}

// voice is the service that speaks a request: the second voice when asked
// for and set up, else the main one.
func (h *Handler) voice(name string) *SpeechService {
	if name == common.AUDIO_VOICE_ALT && h.alt != nil {
		return h.alt
	}
	return h.speech
}

type EmptyInput struct{}

func (h *Handler) Info(_ context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[SpeechInfo]], error) {
	info := h.speech.Info()
	if info.Enabled && h.alt != nil {
		alt := h.alt.Info()
		info.AltBaseURL, info.AltCredit = alt.BaseURL, alt.Credit
	}
	return humautil.OK(info), nil
}

func (h *Handler) Speak(ctx context.Context, input *humautil.Input[SpeechRequest]) (*humautil.Output[humautil.APIResponse[SpeechResponse]], error) {
	who := Requester{UserID: middleware.ContextUserID(ctx), Admin: middleware.ContextIsAdmin(ctx)}
	resp, err := h.voice(input.Body.Voice).Speak(ctx, who, input.Body.Texts)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *Handler) Find(ctx context.Context, input *humautil.Input[SpeechRequest]) (*humautil.Output[humautil.APIResponse[SpeechResponse]], error) {
	resp, err := h.voice(input.Body.Voice).Find(ctx, input.Body.Texts)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
