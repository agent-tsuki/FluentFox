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
	alt *SpeechService
	// cast is each key's voice (main, or a listening character's such as f1).
	cast   *Cast
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

// WithCast speaks each key in the voice the cast gives it.
func (h *Handler) WithCast(cast *Cast) *Handler {
	h.cast = cast
	return h
}

// voice is the service that speaks a request: the second voice when asked
// for and set up, else the cast's voice for the key (main by default).
func (h *Handler) voice(ctx context.Context, name string) *SpeechService {
	if name == common.AUDIO_VOICE_ALT && h.alt != nil {
		return h.alt
	}
	if name == "" {
		name = common.AUDIO_VOICE_MAIN
	}
	if h.cast == nil {
		return h.speech
	}
	return h.cast.Voice(ctx, name)
}

type EmptyInput struct{}

func (h *Handler) Info(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[SpeechInfo]], error) {
	info := h.voice(ctx, common.AUDIO_VOICE_MAIN).Info()
	if info.Enabled && h.alt != nil {
		alt := h.alt.Info()
		info.AltBaseURL, info.AltCredit = alt.BaseURL, alt.Credit
	}
	if info.Enabled && h.cast != nil {
		info.Voices = h.castInfo(ctx)
	}
	return humautil.OK(info), nil
}

func (h *Handler) Speak(ctx context.Context, input *humautil.Input[SpeechRequest]) (*humautil.Output[humautil.APIResponse[SpeechResponse]], error) {
	who := Requester{UserID: middleware.ContextUserID(ctx), Admin: middleware.ContextIsAdmin(ctx)}
	resp, err := h.voice(ctx, input.Body.Voice).Speak(ctx, who, input.Body.Texts)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *Handler) Find(ctx context.Context, input *humautil.Input[SpeechRequest]) (*humautil.Output[humautil.APIResponse[SpeechResponse]], error) {
	resp, err := h.voice(ctx, input.Body.Voice).Find(ctx, input.Body.Texts)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

// castInfo lists the cast keys other than main (whose folder is base_url).
func (h *Handler) castInfo(ctx context.Context) map[string]VoiceInfo {
	voices := map[string]VoiceInfo{}
	for key := range h.cast.Voices(ctx) {
		if key == common.AUDIO_VOICE_MAIN || key == common.AUDIO_VOICE_ALT {
			continue
		}
		if vi := h.cast.Voice(ctx, key).Info(); vi.Enabled {
			voices[key] = VoiceInfo{BaseURL: vi.BaseURL, Credit: vi.Credit}
		}
	}
	if len(voices) == 0 {
		return nil
	}
	return voices
}
