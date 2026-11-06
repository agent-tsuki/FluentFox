package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OpenJTalk runs the open_jtalk binary: the fallback voice when the VOICEVOX
// engine is down. It is a local program, so it keeps working with no network.
type OpenJTalk struct {
	bin   string
	dict  string
	voice string
}

func NewOpenJTalk(bin, dict, voice string) *OpenJTalk {
	return &OpenJTalk{bin: bin, dict: dict, voice: voice}
}

func (o *OpenJTalk) Synthesize(ctx context.Context, text string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "openjtalk-")
	if err != nil {
		return nil, fmt.Errorf("openjtalk: temp dir: %w", err)
	}
	defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup of a private temp dir

	out := filepath.Join(dir, "out.wav")
	// The text goes in on stdin, never as an argument, so it cannot be read as a flag.
	cmd := exec.CommandContext(ctx, o.bin, "-x", o.dict, "-m", o.voice, "-ow", out)
	cmd.Stdin = strings.NewReader(text)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("openjtalk: %w: %s", err, strings.TrimSpace(string(output)))
	}
	wav, err := os.ReadFile(out) //nolint:gosec // path built above inside our own temp dir
	if err != nil {
		return nil, fmt.Errorf("openjtalk: read output: %w", err)
	}
	if !isWAV(wav) {
		return nil, fmt.Errorf("openjtalk: output is not WAV")
	}
	return wav, nil
}
