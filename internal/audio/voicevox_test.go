package audio

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fluentfox/api/internal/common"
)

func TestVoicevoxSynthesize(t *testing.T) {
	var synthesisBody map[string]any
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/audio_query":
			if r.URL.Query().Get("speaker") != "14" || r.URL.Query().Get("text") != "あさ" {
				t.Errorf("audio_query params = %v", r.URL.Query())
			}
			_, _ = io.WriteString(w, `{"accent_phrases":[],"speedScale":1,"outputSamplingRate":44100,"outputStereo":true}`)
		case "/synthesis":
			if err := json.NewDecoder(r.Body).Decode(&synthesisBody); err != nil {
				t.Errorf("decode synthesis body: %v", err)
			}
			_, _ = io.WriteString(w, "RIFF\x00\x00\x00\x00WAVEfmt ")
		default:
			http.NotFound(w, r)
		}
	}))
	defer engine.Close()

	wav, err := NewVoicevox(engine.URL, 14).Synthesize(context.Background(), "あさ")
	if err != nil {
		t.Fatal(err)
	}
	if !isWAV(wav) {
		t.Errorf("result is not WAV: %q", wav)
	}
	if synthesisBody["outputSamplingRate"] != float64(common.AUDIO_SAMPLE_RATE) || synthesisBody["outputStereo"] != false || synthesisBody["speedScale"] != 0.75 {
		t.Errorf("synthesis query = %v, want %d Hz mono at the short-text speed", synthesisBody, common.AUDIO_SAMPLE_RATE)
	}
	if _, ok := synthesisBody["accent_phrases"]; !ok {
		t.Error("engine fields were dropped from the query")
	}
}

func TestVoicevoxEngineError(t *testing.T) {
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer engine.Close()
	if _, err := NewVoicevox(engine.URL, 14).Synthesize(context.Background(), "あ"); err == nil {
		t.Error("err = nil, want the engine's 500")
	}
}
