package audio

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluentfox/api/internal/common"
)

// Synthesizer turns one sentence into WAV bytes.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string) ([]byte, error)
}

// Voice is one way to speak, tried in order: VOICEVOX first, then Open JTalk.
// Profile names the voice's folder in storage, so clips from a fallback voice
// never stand in for the main voice's clips: once the main voice is back, it
// replaces them sentence by sentence.
type Voice struct {
	Profile string
	Credit  string
	Synth   Synthesizer
}

func VoicevoxProfile(speakerID int) string {
	return EngineProfile(common.VOICEVOX_DEFAULT_ENGINE, speakerID)
}

// EngineProfile names a VOICEVOX-compatible engine's voice, e.g.
// "aivisspeech-888753760" for AivisSpeech (same API, other voices), so each
// engine's clips keep their own folder.
func EngineProfile(engine string, speakerID int) string {
	return fmt.Sprintf("%s-%d", engine, speakerID)
}

// OpenJTalkProfile is named after the voice file, e.g.
// ".../nitech_jp_atr503_m001.htsvoice" -> "openjtalk-nitech_jp_atr503_m001".
func OpenJTalkProfile(voicePath string) string {
	name := voicePath[strings.LastIndexAny(voicePath, `/\`)+1:]
	name = strings.TrimSuffix(name, ".htsvoice")
	return "openjtalk-" + strings.ToLower(name)
}
