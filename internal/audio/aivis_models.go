package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
)

// aivisModelsMaxResponseBytes bounds the engine's list of installed models
// (each model carries its voice icons and samples, so it runs to tens of MB).
const aivisModelsMaxResponseBytes = 512 << 20

// EnsureAivisModels installs the AivisHub models an AivisSpeech engine is
// missing, one at a time, so the cast's voices exist after a fresh deploy of
// the engine (its models live on its own disk). It tries every model and
// reports the first failure.
func EnsureAivisModels(ctx context.Context, engineURL string, models []string, log *zap.Logger) error {
	if len(models) == 0 {
		return nil
	}
	client := &http.Client{Timeout: common.AIVIS_MODEL_INSTALL_LIMIT}
	installed, err := installedAivisModels(ctx, client, engineURL)
	if err != nil {
		return err
	}
	var first error
	for _, id := range models {
		if installed[id] {
			log.Info("aivis models: already installed", zap.String("model", id))
			continue
		}
		log.Info("aivis models: installing", zap.String("model", id))
		if err := installAivisModel(ctx, client, engineURL, id); err != nil {
			log.Warn("aivis models: install failed", zap.String("model", id), zap.Error(err))
			if first == nil {
				first = fmt.Errorf("model %s: %w", id, err)
			}
		}
	}
	return first
}

func installedAivisModels(ctx context.Context, client *http.Client, engineURL string) (map[string]bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, engineURL+"/aivm_models", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models: engine answered %d", resp.StatusCode)
	}
	return aivisModelIDs(io.LimitReader(resp.Body, aivisModelsMaxResponseBytes))
}

// aivisModelIDs reads the keys (model UUIDs) of the engine's model list,
// skipping each model's details one at a time instead of holding them all.
func aivisModelIDs(r io.Reader) (map[string]bool, error) {
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	if tok != json.Delim('{') {
		return nil, fmt.Errorf("decode models: expected an object, got %v", tok)
	}
	out := map[string]bool{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode models: %w", err)
		}
		id, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("decode models: unexpected key %v", tok)
		}
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, fmt.Errorf("decode models: %w", err)
		}
		out[id] = true
	}
	return out, nil
}

func installAivisModel(ctx context.Context, client *http.Client, engineURL, id string) error {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("url", fmt.Sprintf(common.AIVIS_HUB_MODEL_URL, id)); err != nil {
		return fmt.Errorf("build form: %w", err)
	}
	if err := form.Close(); err != nil {
		return fmt.Errorf("build form: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, engineURL+"/aivm_models/install", &body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("install: engine answered %d", resp.StatusCode)
	}
	return nil
}
