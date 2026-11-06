package audio

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

const testCDN = "https://cdn.example"

func mainKey(sentence string) string {
	return common.AUDIO_OBJECT_PREFIX + "voicevox-14/" + clipHash(sentence) + ".wav"
}

func newTestService(store ObjectStore, voices ...Voice) *SpeechService {
	return NewSpeechService(store, testCDN+"/", voices, WAVPassthrough{}, zap.NewNop())
}

func mainVoice(s Synthesizer) Voice {
	return Voice{Profile: "voicevox-14", Credit: "VOICEVOX:冥鳴ひまり", Synth: s}
}

func TestSpeak(t *testing.T) {
	user := Requester{UserID: uuid.New()}
	ctx := context.Background()

	t.Run("stored clip is returned without synthesis", func(t *testing.T) {
		synth := &fakeSynth{}
		svc := newTestService(newFakeStore(mainKey("あさ")), mainVoice(synth))
		resp, err := svc.Speak(ctx, user, []string{" あさ "})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := resp.Items[0].Segments[0].URL, testCDN+"/"+mainKey("あさ"); got != want {
			t.Errorf("url = %s, want %s", got, want)
		}
		if synth.count() != 0 {
			t.Errorf("synthesized %d times, want 0", synth.count())
		}
		if len(resp.Credits) != 1 || resp.Credits[0] != "VOICEVOX:冥鳴ひまり" || resp.Pending {
			t.Errorf("credits = %v, pending = %v", resp.Credits, resp.Pending)
		}
	})

	t.Run("missing clip is generated once, then served from memory", func(t *testing.T) {
		store, synth := newFakeStore(), &fakeSynth{}
		svc := newTestService(store, mainVoice(synth))
		for range 2 {
			if _, err := svc.Speak(ctx, user, []string{"ねこです。いぬです。"}); err != nil {
				t.Fatal(err)
			}
		}
		if synth.count() != 2 || store.puts != 2 {
			t.Errorf("synth calls = %d, puts = %d; want 2 each", synth.count(), store.puts)
		}
		if store.heads != 2 {
			t.Errorf("storage lookups = %d, want 2 (second request served from memory)", store.heads)
		}
	})

	t.Run("a sentence shared by two texts is resolved once", func(t *testing.T) {
		synth := &fakeSynth{}
		svc := newTestService(newFakeStore(), mainVoice(synth))
		resp, err := svc.Speak(ctx, user, []string{"はい。", "はい。いいえ。"})
		if err != nil {
			t.Fatal(err)
		}
		if synth.count() != 2 || len(resp.Items[1].Segments) != 2 {
			t.Errorf("synth calls = %d, segments = %d", synth.count(), len(resp.Items[1].Segments))
		}
	})

	t.Run("fallback voice speaks when the main one fails, and the main one rests", func(t *testing.T) {
		main, fallback := &fakeSynth{fail: true}, &fakeSynth{}
		svc := newTestService(newFakeStore(), mainVoice(main),
			Voice{Profile: "openjtalk-mei_normal", Credit: "HTS Voice Mei", Synth: fallback})
		resp, err := svc.Speak(ctx, user, []string{"あ"})
		if err != nil {
			t.Fatal(err)
		}
		seg := resp.Items[0].Segments[0]
		if seg.Voice != "openjtalk-mei_normal" || !strings.Contains(seg.URL, "/openjtalk-mei_normal/") {
			t.Errorf("segment = %+v, want the fallback voice", seg)
		}
		if _, err := svc.Speak(ctx, user, []string{"い"}); err != nil {
			t.Fatal(err)
		}
		if main.count() != 1 {
			t.Errorf("main voice tried %d times, want 1 (skipped while resting)", main.count())
		}
	})

	t.Run("generations past the per-request budget come back pending", func(t *testing.T) {
		svc := newTestService(newFakeStore(), mainVoice(&fakeSynth{}))
		texts := make([]string, common.AUDIO_MAX_NEW_PER_REQUEST+2)
		for i := range texts {
			texts[i] = fmt.Sprintf("ことば%d", i)
		}
		resp, err := svc.Speak(ctx, user, texts)
		if err != nil {
			t.Fatal(err)
		}
		ready := 0
		for _, item := range resp.Items {
			if item.Segments[0].URL != "" {
				ready++
			}
		}
		if ready != common.AUDIO_MAX_NEW_PER_REQUEST || !resp.Pending {
			t.Errorf("ready = %d, pending = %v; want %d and true", ready, resp.Pending, common.AUDIO_MAX_NEW_PER_REQUEST)
		}
	})

	t.Run("hourly limit defers new clips", func(t *testing.T) {
		svc := newTestService(newFakeStore(), mainVoice(&fakeSynth{}))
		svc.limiter = newGenerationLimiter(1, svc.limiter.window)
		resp, err := svc.Speak(ctx, user, []string{"ひとつ", "ふたつ"})
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Pending {
			t.Error("pending = false, want true once the limit is reached")
		}
	})

	t.Run("admins are not held to the hourly limit", func(t *testing.T) {
		svc := newTestService(newFakeStore(), mainVoice(&fakeSynth{}))
		svc.limiter = newGenerationLimiter(1, svc.limiter.window)
		admin := Requester{UserID: uuid.New(), Admin: true}
		resp, err := svc.Speak(ctx, admin, []string{"ひとつ", "ふたつ", "みっつ"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Pending {
			t.Error("pending = true, want every clip made for an admin")
		}
	})
}

func TestSpeakErrors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		svc   *SpeechService
		texts []string
		want  error
	}{
		{"no storage is unavailable", newTestService(nil, mainVoice(&fakeSynth{})), []string{"あ"}, exceptions.ErrAudioUnavailable},
		{"no voice is unavailable", newTestService(newFakeStore()), []string{"あ"}, exceptions.ErrAudioUnavailable},
		{"blank text", newTestService(newFakeStore(), mainVoice(&fakeSynth{})), []string{"　 "}, exceptions.ErrAudioTextEmpty},
		{"text too long", newTestService(newFakeStore(), mainVoice(&fakeSynth{})), []string{strings.Repeat("あ", common.AUDIO_MAX_TEXT_RUNES+1)}, exceptions.ErrAudioTextTooLong},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.svc.Speak(ctx, Requester{UserID: uuid.New()}, tc.texts); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	info := newTestService(newFakeStore(), mainVoice(&fakeSynth{})).Info()
	if !info.Enabled || info.BaseURL != testCDN+"/"+common.AUDIO_OBJECT_PREFIX+"voicevox-14" || info.Ext != ".wav" {
		t.Errorf("info = %+v", info)
	}
	if off := newTestService(nil).Info(); off.Enabled {
		t.Errorf("info without storage = %+v, want disabled", off)
	}
}

type slowSynth struct{ release chan struct{} }

func (s *slowSynth) Synthesize(ctx context.Context, text string) ([]byte, error) {
	select {
	case <-s.release:
		return []byte("RIFF....WAVE" + text), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// A queue longer than the slot wait defers the extra clips; it must not mark
// the voice as down, or the next request would skip a healthy voice.
func TestBusyVoiceDefersWithoutMarkingItDown(t *testing.T) {
	synth := &slowSynth{release: make(chan struct{})}
	svc := newTestService(newFakeStore(), mainVoice(synth))
	svc.slots = make(chan struct{}) // no free slot at all
	resp, err := svc.Speak(context.Background(), Requester{UserID: uuid.New()}, []string{"あ"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Pending || resp.Items[0].Segments[0].URL != "" {
		t.Errorf("resp = %+v, want the clip deferred", resp)
	}
	if svc.downUntil[0].Load() != 0 {
		t.Error("voice was marked down for being busy")
	}
}
