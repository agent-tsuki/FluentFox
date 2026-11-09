package audio

import (
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/cache"
)

// CastStore reads the voice cast; *Repository satisfies it.
type CastStore interface {
	ListCastVoices(ctx context.Context) ([]CastVoice, error)
}

// SynthFor makes the engine voice for a style, e.g. NewVoicevox(url, id).
type SynthFor func(styleID int) Synthesizer

const castCacheKey = "cast"

// Cast is the voice each key speaks with, as a super-admin set it in
// audio_voice. A key's voice is only its engine style, with no fallback: when
// a voice changes, its clips do not exist yet and clients say so instead of
// playing another voice. A key with no row is spoken by the main voice from
// the env. Services are kept per voice folder, so the known-clips cache and
// a voice's down state survive the cast being read again.
type Cast struct {
	main     *SpeechService
	store    CastStore
	synth    SynthFor
	rows     atomic.Pointer[cache.TTL[map[string]CastVoice]]
	mu       sync.Mutex
	services map[string]*SpeechService
	log      *zap.Logger
}

// NewCast takes the env's main voice service; synth is nil when there is no engine.
func NewCast(main *SpeechService, store CastStore, synth SynthFor, log *zap.Logger) *Cast {
	c := &Cast{main: main, store: store, synth: synth, services: map[string]*SpeechService{}, log: log}
	c.Invalidate()
	return c
}

// Invalidate makes the next request read the cast again (after a change on
// this server; other servers see it within common.AUDIO_CAST_TTL).
func (c *Cast) Invalidate() {
	c.rows.Store(cache.NewTTL[map[string]CastVoice](common.AUDIO_CAST_TTL, 1))
}

// Voices is the cast by key. A database error leaves every key on the main voice.
func (c *Cast) Voices(ctx context.Context) map[string]CastVoice {
	rows, err := c.rows.Load().Get(ctx, castCacheKey, func(ctx context.Context) (map[string]CastVoice, error) {
		list, err := c.store.ListCastVoices(ctx)
		if err != nil {
			return nil, err //nolint:wrapcheck // the repository wraps it
		}
		byKey := make(map[string]CastVoice, len(list))
		for i := range list {
			byKey[list[i].Key] = list[i]
		}
		return byKey, nil
	})
	if err != nil {
		c.log.Warn("audio cast unavailable; using the main voice", zap.Error(err))
		return nil
	}
	return rows
}

// Voice is the service that speaks key.
func (c *Cast) Voice(ctx context.Context, key string) *SpeechService {
	row, ok := c.Voices(ctx)[key]
	if !ok || c.synth == nil {
		return c.main
	}
	return c.service(&row)
}

func (c *Cast) service(row *CastVoice) *SpeechService {
	profile := row.Profile()
	c.mu.Lock()
	defer c.mu.Unlock()
	svc, ok := c.services[profile]
	if !ok || svc.voices[0].Credit != row.Credit {
		svc = c.main.WithVoices([]Voice{{Profile: profile, Credit: row.Credit, Synth: c.synth(row.StyleID)}})
		c.services[profile] = svc
	}
	return svc
}
