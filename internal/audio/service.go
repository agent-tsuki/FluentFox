package audio

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/cache"
	"github.com/fluentfox/api/pkg/exceptions"
)

// ObjectStore keeps the clips; *storage.R2Store satisfies it.
type ObjectStore interface {
	Exists(ctx context.Context, key string) (bool, error)
	Put(ctx context.Context, key, contentType, cacheControl string, body []byte) error
}

// ClipIndex records the stored clips (*Repository satisfies it), so finding a
// clip is one database query per request instead of a storage request per
// sentence.
type ClipIndex interface {
	ExistingClips(ctx context.Context, keys []string) (map[string]bool, error)
	RecordClip(ctx context.Context, c *ClipRecord) error
}

// Requester is who asked for audio. Admins pre-generate whole courses, so the
// per-user hourly limit does not apply to them; the per-request cap still does,
// which is what protects the voice engine.
type Requester struct {
	UserID uuid.UUID
	Admin  bool
	// LookupOnly finds stored clips and never generates (anyone may ask).
	LookupOnly bool
}

// errDeferred means a clip was not generated now (request budget or the
// user's hourly limit); it is returned without a URL for the client to ask again.
var errDeferred = errors.New("audio: generation deferred")

// SpeechService answers "where is the audio for these texts?". Each sentence's
// clip is looked up in storage and generated only if missing, then served from
// the CDN for good. Clips known to exist are remembered in memory, so most
// requests touch neither storage nor a voice.
type SpeechService struct {
	store     ObjectStore
	publicURL string
	voices    []Voice
	encoder   Encoder
	known     *cache.TTL[string]
	clips     ClipIndex
	limiter   *generationLimiter
	slots     chan struct{}
	downUntil []atomic.Int64
	log       *zap.Logger
}

// NewSpeechService takes the voices in order of preference. A nil store or no
// voices leaves the service off (503), and clients use the browser's voice.
func NewSpeechService(store ObjectStore, publicURL string, voices []Voice, encoder Encoder, log *zap.Logger) *SpeechService {
	return &SpeechService{
		store:     store,
		publicURL: strings.TrimRight(publicURL, "/"),
		voices:    voices,
		encoder:   encoder,
		known:     cache.NewTTL[string](common.AUDIO_KNOWN_CLIPS_TTL, common.AUDIO_KNOWN_CLIPS_MAX),
		limiter:   newGenerationLimiter(common.AUDIO_GENERATIONS_PER_HOUR, time.Hour),
		slots:     make(chan struct{}, common.AUDIO_MAX_CONCURRENT_SYNTH),
		downUntil: make([]atomic.Int64, len(voices)),
		log:       log,
	}
}

// WithVoices is the same service speaking with other voices (a listening
// character's). It shares the storage, the engine slots, the hourly limit and
// the known-clips cache, so more voices do not mean more load on the engine.
func (s *SpeechService) WithVoices(voices []Voice) *SpeechService {
	return &SpeechService{
		store:     s.store,
		publicURL: s.publicURL,
		voices:    voices,
		encoder:   s.encoder,
		known:     s.known,
		clips:     s.clips,
		limiter:   s.limiter,
		slots:     s.slots,
		downUntil: make([]atomic.Int64, len(voices)),
		log:       s.log,
	}
}

// WithClipIndex looks clips up in (and records them to) the index before
// storage. Call it before WithVoices, which shares it.
func (s *SpeechService) WithClipIndex(clips ClipIndex) *SpeechService {
	s.clips = clips
	return s
}

func (s *SpeechService) objectKey(voice Voice, sentence string) string {
	return common.AUDIO_OBJECT_PREFIX + voice.Profile + "/" + clipHash(sentence) + s.encoder.Ext()
}

// indexedClips asks the index once for every clip the request could use. An
// index error is not fatal: storage still answers.
func (s *SpeechService) indexedClips(ctx context.Context, split [][]string) map[string]bool {
	if s.clips == nil {
		return nil
	}
	var keys []string
	for _, sentences := range split {
		for _, sentence := range sentences {
			for _, voice := range s.voices {
				keys = append(keys, s.objectKey(voice, sentence))
			}
		}
	}
	found, err := s.clips.ExistingClips(ctx, keys)
	if err != nil {
		s.log.Warn("audio clip index unavailable; asking storage", zap.Error(err))
		return nil
	}
	return found
}

func (s *SpeechService) recordClip(ctx context.Context, who Requester, voice Voice, key, sentence string, size int) {
	if s.clips == nil {
		return
	}
	rec := &ClipRecord{ObjectKey: key, Profile: voice.Profile, Text: sentence, Bytes: size}
	if who.UserID != uuid.Nil {
		rec.CreatedBy = &who.UserID
	}
	if err := s.clips.RecordClip(ctx, rec); err != nil {
		s.log.Warn("audio: record clip", zap.String("key", key), zap.Error(err))
	}
}

func (s *SpeechService) enabled() bool {
	return s.store != nil && len(s.voices) > 0
}

// Info tells clients whether spoken audio is on, the main voice's folder (to
// name their saved copies and drop ones from an older folder) and its credit.
func (s *SpeechService) Info() SpeechInfo {
	if !s.enabled() {
		return SpeechInfo{}
	}
	main := s.voices[0]
	return SpeechInfo{
		Enabled: true,
		BaseURL: s.publicURL + "/" + common.AUDIO_OBJECT_PREFIX + main.Profile,
		Ext:     s.encoder.Ext(),
		Credit:  main.Credit,
	}
}

// Find returns the CDN URLs of the clips already stored for texts, without
// generating any; a sentence with no clip has no url. Clients fetch only
// URLs listed here or by Speak, so the CDN is never asked for a clip that
// does not exist (it would keep that 404 for hours).
func (s *SpeechService) Find(ctx context.Context, texts []string) (SpeechResponse, error) {
	return s.Speak(ctx, Requester{LookupOnly: true}, texts)
}

func (s *SpeechService) Speak(ctx context.Context, who Requester, texts []string) (SpeechResponse, error) {
	if !s.enabled() {
		return SpeechResponse{}, exceptions.ErrAudioUnavailable
	}
	split := make([][]string, len(texts))
	for i, text := range texts {
		if utf8.RuneCountInString(text) > common.AUDIO_MAX_TEXT_RUNES {
			return SpeechResponse{}, exceptions.ErrAudioTextTooLong
		}
		if split[i] = Sentences(text); len(split[i]) == 0 {
			return SpeechResponse{}, exceptions.ErrAudioTextEmpty
		}
	}
	segments, err := s.resolveAll(ctx, who, split)
	if err != nil {
		return SpeechResponse{}, err
	}
	return s.buildResponse(split, segments), nil
}

// resolveAll resolves each distinct sentence once, a few at a time.
func (s *SpeechService) resolveAll(ctx context.Context, who Requester, split [][]string) (map[string]SpeechSegment, error) {
	budget := &atomic.Int32{}
	budget.Store(int32(common.AUDIO_MAX_NEW_PER_REQUEST))
	indexed := s.indexedClips(ctx, split)
	var mu sync.Mutex
	segments := make(map[string]SpeechSegment)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(common.AUDIO_MAX_PARALLEL_LOOKUPS)
	for _, sentences := range split {
		for _, sentence := range sentences {
			mu.Lock()
			_, seen := segments[sentence]
			segments[sentence] = SpeechSegment{Text: sentence}
			mu.Unlock()
			if seen {
				continue
			}
			g.Go(func() error {
				seg := s.resolve(gctx, who, sentence, budget, indexed)
				mu.Lock()
				segments[sentence] = seg
				mu.Unlock()
				return nil
			})
		}
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("audio: resolve sentences: %w", err)
	}
	return segments, nil
}

// resolve tries each voice in order: its stored clip, else a new one. A voice
// that just failed is skipped for a minute so a dead engine does not add its
// timeout to every request.
func (s *SpeechService) resolve(ctx context.Context, who Requester, sentence string, budget *atomic.Int32, indexed map[string]bool) SpeechSegment {
	for i, voice := range s.voices {
		if time.Now().UnixNano() < s.downUntil[i].Load() {
			continue
		}
		key := s.objectKey(voice, sentence)
		url, err := s.known.Get(ctx, key, func(ctx context.Context) (string, error) {
			if indexed[key] {
				return s.publicURL + "/" + key, nil
			}
			return s.findOrCreate(ctx, who, voice, key, sentence, budget)
		})
		if err == nil {
			return SpeechSegment{Text: sentence, URL: url, Voice: voice.Profile, credit: voice.Credit}
		}
		if errors.Is(err, errDeferred) {
			if who.LookupOnly {
				continue // The next voice may have spoken it.
			}
			return SpeechSegment{Text: sentence}
		}
		s.downUntil[i].Store(time.Now().Add(common.AUDIO_VOICE_RETRY_AFTER).UnixNano())
		s.log.Warn("audio voice failed; trying the next one", zap.String("voice", voice.Profile), zap.Error(err))
	}
	return SpeechSegment{Text: sentence}
}

func (s *SpeechService) findOrCreate(
	ctx context.Context, who Requester, voice Voice, key, sentence string, budget *atomic.Int32,
) (string, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, common.AUDIO_STORE_TIMEOUT)
	defer cancel()
	exists, err := s.store.Exists(lookupCtx, key)
	if err != nil {
		return "", fmt.Errorf("audio: look up clip: %w", err)
	}
	if exists {
		// Made before the index existed: add it, so the next lookup skips storage.
		s.recordClip(ctx, Requester{}, voice, key, sentence, 0)
		return s.publicURL + "/" + key, nil
	}
	if who.LookupOnly || budget.Add(-1) < 0 || (!who.Admin && !s.limiter.take(who.UserID)) {
		return "", errDeferred
	}
	clip, err := s.generate(ctx, voice, sentence)
	if err != nil {
		return "", err
	}
	storeCtx, cancelStore := context.WithTimeout(ctx, common.AUDIO_STORE_TIMEOUT)
	defer cancelStore()
	if err := s.store.Put(storeCtx, key, s.encoder.ContentType(), common.AUDIO_CACHE_CONTROL, clip); err != nil {
		return "", fmt.Errorf("audio: store clip: %w", err)
	}
	s.recordClip(ctx, who, voice, key, sentence, len(clip))
	return s.publicURL + "/" + key, nil
}

// generate runs a voice and the encoder. Only a few run at once: they are the
// expensive part, and the VOICEVOX engine renders one sentence per CPU.
// Waiting for a slot is not the voice failing: a clip that waits too long is
// deferred (errDeferred), so a busy queue never marks a healthy voice as down.
func (s *SpeechService) generate(ctx context.Context, voice Voice, sentence string) ([]byte, error) {
	waitCtx, cancelWait := context.WithTimeout(ctx, common.AUDIO_SLOT_WAIT)
	defer cancelWait()
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-waitCtx.Done():
		return nil, fmt.Errorf("audio: all voices busy: %w", errDeferred)
	}
	synthCtx, cancel := context.WithTimeout(ctx, common.AUDIO_SYNTH_TIMEOUT)
	defer cancel()
	wav, err := voice.Synth.Synthesize(synthCtx, sentence)
	if err != nil {
		return nil, fmt.Errorf("audio: synthesize with %s: %w", voice.Profile, err)
	}
	clip, err := s.encoder.Encode(synthCtx, normalizePeak(wav))
	if err != nil {
		return nil, fmt.Errorf("audio: encode: %w", err)
	}
	return clip, nil
}

func (s *SpeechService) buildResponse(split [][]string, segments map[string]SpeechSegment) SpeechResponse {
	resp := SpeechResponse{Items: make([]SpeechItem, len(split)), Credits: []string{}}
	credits := map[string]bool{}
	for i, sentences := range split {
		item := SpeechItem{Segments: make([]SpeechSegment, len(sentences))}
		for j, sentence := range sentences {
			seg := segments[sentence]
			item.Segments[j] = seg
			if seg.URL == "" {
				resp.Pending = true
			} else if !credits[seg.credit] {
				credits[seg.credit] = true
				resp.Credits = append(resp.Credits, seg.credit)
			}
		}
		resp.Items[i] = item
	}
	return resp
}
