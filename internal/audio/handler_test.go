package audio

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/humautil"
)

func altVoice(s Synthesizer) Voice {
	return Voice{Profile: "voicevox-13", Credit: "VOICEVOX:青山龍星", Synth: s} //nolint:gosec // a credit line, not a secret
}

func speechInput(voice string, texts ...string) *humautil.Input[SpeechRequest] {
	return &humautil.Input[SpeechRequest]{Body: SpeechRequest{Texts: texts, Voice: voice}}
}

func TestHandlerAltVoice(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	main := newTestService(store, mainVoice(&fakeSynth{}))

	t.Run("without a second voice, alt is spoken by the main voice", func(t *testing.T) {
		h := NewHandler(main, zap.NewNop()).WithAltVoice(nil)
		out, err := h.Find(ctx, speechInput(common.AUDIO_VOICE_ALT, "はい"))
		if err != nil {
			t.Fatal(err)
		}
		if out.Body.Data.Items[0].Segments[0].Text != "はい" {
			t.Fatalf("unexpected response %+v", out.Body.Data)
		}
		info, _ := h.Info(ctx, &EmptyInput{})
		if info.Body.Data.AltBaseURL != "" {
			t.Fatalf("alt_base_url = %q, want none", info.Body.Data.AltBaseURL)
		}
	})

	t.Run("alt uses the second voice's own folder", func(t *testing.T) {
		alt := newTestService(newFakeStore(common.AUDIO_OBJECT_PREFIX+"voicevox-13/"+clipHash("はい")+".wav"), altVoice(&fakeSynth{}))
		h := NewHandler(main, zap.NewNop()).WithAltVoice(alt)
		out, err := h.Find(ctx, speechInput(common.AUDIO_VOICE_ALT, "はい"))
		if err != nil {
			t.Fatal(err)
		}
		if url := out.Body.Data.Items[0].Segments[0].URL; !strings.Contains(url, "voicevox-13/") {
			t.Fatalf("url = %q, want the second voice's folder", url)
		}
		main, err := h.Find(ctx, speechInput("", "はい"))
		if err != nil {
			t.Fatal(err)
		}
		if main.Body.Data.Items[0].Segments[0].URL != "" {
			t.Fatalf("main voice answered with the second voice's clip")
		}
		info, _ := h.Info(ctx, &EmptyInput{})
		if !strings.HasSuffix(info.Body.Data.AltBaseURL, "voicevox-13") || info.Body.Data.AltCredit != "VOICEVOX:青山龍星" {
			t.Fatalf("info = %+v", info.Body.Data)
		}
	})
}
