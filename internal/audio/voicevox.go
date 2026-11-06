package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/fluentfox/api/internal/common"
)

// voicevoxMaxResponseBytes bounds a reply from the engine: a 300-character
// sentence at 24 kHz mono is well under 2 MB of WAV.
const voicevoxMaxResponseBytes = 8 << 20

// Voicevox talks to a VOICEVOX engine over HTTP: /audio_query builds the
// reading and intonation, /synthesis renders it.
type Voicevox struct {
	baseURL string
	speaker string
	client  *http.Client
}

func NewVoicevox(baseURL string, speakerID int) *Voicevox {
	return &Voicevox{
		baseURL: baseURL,
		speaker: strconv.Itoa(speakerID),
		client:  &http.Client{Timeout: common.AUDIO_SYNTH_TIMEOUT},
	}
}

func (v *Voicevox) Synthesize(ctx context.Context, text string) ([]byte, error) {
	query, err := v.post(ctx, "/audio_query", url.Values{"speaker": {v.speaker}, "text": {text}}, nil)
	if err != nil {
		return nil, err
	}
	// Keep every field the engine returned; only the output format changes.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(query, &fields); err != nil {
		return nil, fmt.Errorf("voicevox: decode audio query: %w", err)
	}
	fields["outputSamplingRate"] = json.RawMessage(strconv.Itoa(common.AUDIO_SAMPLE_RATE))
	fields["outputStereo"] = json.RawMessage("false")
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("voicevox: encode audio query: %w", err)
	}
	wav, err := v.post(ctx, "/synthesis", url.Values{"speaker": {v.speaker}}, body)
	if err != nil {
		return nil, err
	}
	if !isWAV(wav) {
		return nil, fmt.Errorf("voicevox: synthesis did not return WAV")
	}
	return wav, nil
}

func (v *Voicevox) post(ctx context.Context, path string, params url.Values, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+path+"?"+params.Encode(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("voicevox: build %s request: %w", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voicevox: %s: %w", path, err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	data, err := io.ReadAll(io.LimitReader(resp.Body, voicevoxMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("voicevox: read %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("voicevox: %s answered %d", path, resp.StatusCode)
	}
	return data, nil
}

func isWAV(b []byte) bool {
	return len(b) > 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WAVE"
}
