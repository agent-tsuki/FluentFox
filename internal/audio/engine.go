package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

// engineMaxResponseBytes bounds /speakers and /version replies.
const engineMaxResponseBytes = 8 << 20

// EngineSpeaker is one voice the engine has, with its styles (/speakers).
type EngineSpeaker struct {
	Name        string        `json:"name"`
	SpeakerUUID string        `json:"speaker_uuid"`
	Styles      []EngineStyle `json:"styles"`
}

type EngineStyle struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Engine is the VOICEVOX-compatible engine behind VOICEVOX_URL, as the admin
// page sees it.
type Engine struct {
	baseURL string
	name    string
	client  *http.Client
}

func NewEngine(baseURL, name string) *Engine {
	return &Engine{baseURL: strings.TrimRight(baseURL, "/"), name: name, client: &http.Client{Timeout: common.AUDIO_ENGINE_TIMEOUT}}
}

func (e *Engine) Name() string { return e.name }

func (e *Engine) Configured() bool { return e != nil && e.baseURL != "" }

func (e *Engine) Version(ctx context.Context) (string, error) {
	var version string
	if err := e.getJSON(ctx, e.client, "/version", &version); err != nil {
		return "", err
	}
	return version, nil
}

func (e *Engine) Speakers(ctx context.Context) ([]EngineSpeaker, error) {
	var speakers []EngineSpeaker
	if err := e.getJSON(ctx, e.client, "/speakers", &speakers); err != nil {
		return nil, err
	}
	return speakers, nil
}

// Synth speaks with one style, as the API does.
func (e *Engine) Synth(styleID int) Synthesizer {
	return NewVoicevox(e.baseURL, styleID)
}

// ModelOf finds the installed AivisHub model a speaker comes from, so a fresh
// engine can install it again. Nil for an engine without models (VOICEVOX).
func (e *Engine) ModelOf(ctx context.Context, speakerUUID string) (*uuid.UUID, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+"/aivm_models", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("engine: build request: %w", err)
	}
	resp, err := (&http.Client{Timeout: common.AUDIO_ENGINE_MODELS_LIMIT}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine: list models: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("engine: list models: engine answered %d", resp.StatusCode)
	}
	return modelOfSpeaker(io.LimitReader(resp.Body, aivisModelsMaxResponseBytes), speakerUUID)
}

// aivisModel is the part of an /aivm_models entry that names its speakers.
type aivisModel struct {
	Speakers []struct {
		Speaker struct {
			SpeakerUUID string `json:"speaker_uuid"`
		} `json:"speaker"`
	} `json:"speakers"`
}

// modelOfSpeaker reads the model list one model at a time (it carries icons
// and samples, tens of MB in all).
func modelOfSpeaker(r io.Reader, speakerUUID string) (*uuid.UUID, error) {
	dec := json.NewDecoder(r)
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("engine: decode models: %w", err)
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("engine: decode models: %w", err)
		}
		var model aivisModel
		if err := dec.Decode(&model); err != nil {
			return nil, fmt.Errorf("engine: decode models: %w", err)
		}
		for _, s := range model.Speakers {
			if s.Speaker.SpeakerUUID != speakerUUID {
				continue
			}
			key, _ := tok.(string)
			id, err := uuid.Parse(key)
			if err != nil {
				return nil, fmt.Errorf("engine: model id %q: %w", key, err)
			}
			return &id, nil
		}
	}
	return nil, nil
}

func (e *Engine) getJSON(ctx context.Context, client *http.Client, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+path, http.NoBody)
	if err != nil {
		return fmt.Errorf("engine: build %s request: %w", path, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("engine: %s: %w", path, err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("engine: %s answered %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, engineMaxResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("engine: decode %s: %w", path, err)
	}
	return nil
}
