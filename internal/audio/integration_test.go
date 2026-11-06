package audio

import (
	"context"
	"os"
	"testing"
)

// Runs the real fallback chain (open_jtalk, then ffmpeg) when its binaries are
// configured, e.g. OPEN_JTALK_BIN, OPEN_JTALK_DICT, OPEN_JTALK_VOICE and
// FFMPEG_BIN as in the server's env. Skipped otherwise. AUDIO_TEST_OUT, when
// set, keeps the MP3 to listen to.
func TestOpenJTalkToMP3(t *testing.T) {
	bin, dict, voice, ffmpeg := os.Getenv("OPEN_JTALK_BIN"), os.Getenv("OPEN_JTALK_DICT"), os.Getenv("OPEN_JTALK_VOICE"), os.Getenv("FFMPEG_BIN")
	if bin == "" || dict == "" || voice == "" || ffmpeg == "" {
		t.Skip("set OPEN_JTALK_BIN, OPEN_JTALK_DICT, OPEN_JTALK_VOICE and FFMPEG_BIN to run")
	}
	ctx := context.Background()
	wav, err := NewOpenJTalk(bin, dict, voice).Synthesize(ctx, "あさです。")
	if err != nil {
		t.Fatal(err)
	}
	mp3, err := NewFFmpegMP3(ffmpeg).Encode(ctx, wav)
	if err != nil {
		t.Fatal(err)
	}
	if len(mp3) == 0 || len(mp3) >= len(wav) {
		t.Errorf("mp3 is %d bytes from %d bytes of WAV, want smaller and non-empty", len(mp3), len(wav))
	}
	t.Logf("wav %d bytes -> mp3 %d bytes", len(wav), len(mp3))
	if out := os.Getenv("AUDIO_TEST_OUT"); out != "" {
		if err := os.WriteFile(out, mp3, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
