package audio

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/fluentfox/api/internal/common"
)

// Encoder prepares a WAV clip for storage. The file extension is part of the
// clip's URL, so changing encoders stores new files rather than mixing formats.
type Encoder interface {
	Encode(ctx context.Context, wav []byte) ([]byte, error)
	Ext() string
	ContentType() string
}

// FFmpegMP3 compresses to mono MP3, about a tenth of the WAV size, through
// the ffmpeg binary (a separate program, so its licence stays separate too).
type FFmpegMP3 struct {
	bin string
}

func NewFFmpegMP3(bin string) *FFmpegMP3 {
	return &FFmpegMP3{bin: bin}
}

func (f *FFmpegMP3) Encode(ctx context.Context, wav []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, f.bin, //nolint:gosec // fixed arguments; the binary comes from config
		"-hide_banner", "-loglevel", "error",
		"-f", "wav", "-i", "pipe:0",
		"-ac", "1", "-ar", strconv.Itoa(common.AUDIO_SAMPLE_RATE),
		"-c:a", "libmp3lame", "-b:a", common.AUDIO_MP3_BITRATE,
		"-f", "mp3", "pipe:1",
	)
	cmd.Stdin = bytes.NewReader(wav)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

func (f *FFmpegMP3) Ext() string         { return ".mp3" }
func (f *FFmpegMP3) ContentType() string { return "audio/mpeg" }

// WAVPassthrough stores clips as the synthesizer made them, for hosts without ffmpeg.
type WAVPassthrough struct{}

func (WAVPassthrough) Encode(_ context.Context, wav []byte) ([]byte, error) { return wav, nil }
func (WAVPassthrough) Ext() string                                          { return ".wav" }
func (WAVPassthrough) ContentType() string                                  { return "audio/wav" }
