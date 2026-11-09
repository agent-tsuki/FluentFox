package audio

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

// SuperAdmins reads the admin flag from the database (not the token), so a
// revoked admin loses access at once; *access.Checker satisfies it.
type SuperAdmins interface {
	IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

func requireSuperAdmin(ctx context.Context, admins SuperAdmins, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return exceptions.ErrForbidden
	}
	ok, err := admins.IsSuperAdmin(ctx, userID)
	if err != nil {
		return fmt.Errorf("audio admin: %w", err)
	}
	if !ok {
		return exceptions.ErrForbidden
	}
	return nil
}

// EngineAPI is the voice engine as the admin page uses it; *Engine satisfies it.
type EngineAPI interface {
	Name() string
	Configured() bool
	Version(ctx context.Context) (string, error)
	Speakers(ctx context.Context) ([]EngineSpeaker, error)
	ModelOf(ctx context.Context, speakerUUID string) (*uuid.UUID, error)
	Synth(styleID int) Synthesizer
}

// TextSource lists what the course speaks; *Repository satisfies it.
type TextSource interface {
	CourseTexts(ctx context.Context) ([]CourseText, error)
}

// VoiceListService shows each voice key with how many of its clips exist.
type VoiceListService struct {
	admins SuperAdmins
	cast   *Cast
	texts  TextSource
	clips  ClipIndex
	log    *zap.Logger
}

func NewVoiceListService(admins SuperAdmins, cast *Cast, texts TextSource, clips ClipIndex, log *zap.Logger) *VoiceListService {
	return &VoiceListService{admins: admins, cast: cast, texts: texts, clips: clips, log: log}
}

func (s *VoiceListService) List(ctx context.Context, userID uuid.UUID) (AdminVoicesResponse, error) {
	if err := requireSuperAdmin(ctx, s.admins, userID); err != nil {
		return AdminVoicesResponse{}, err
	}
	texts, err := s.texts.CourseTexts(ctx)
	if err != nil {
		return AdminVoicesResponse{}, fmt.Errorf("audio admin: %w", err)
	}
	rows := s.cast.Voices(ctx)
	byKey := map[string][]string{common.AUDIO_VOICE_MAIN: nil}
	for key := range rows {
		byKey[key] = nil
	}
	for _, t := range texts {
		byKey[t.Voice] = append(byKey[t.Voice], t.Text)
	}
	voices := make([]AdminVoice, 0, len(byKey))
	var keys []string
	for key, keyTexts := range byKey {
		v, clipKeys := s.voiceOf(ctx, key, rows, keyTexts)
		voices = append(voices, v)
		keys = append(keys, clipKeys...)
	}
	stored, err := s.clips.ExistingClips(ctx, keys)
	if err != nil {
		return AdminVoicesResponse{}, fmt.Errorf("audio admin: %w", err)
	}
	for i := range voices {
		voices[i].Ready = countStored(ctx, s.cast, voices[i].Key, byKey[voices[i].Key], stored)
	}
	sort.Slice(voices, func(i, j int) bool { return voiceOrder(voices[i].Key) < voiceOrder(voices[j].Key) })
	return AdminVoicesResponse{Voices: voices}, nil
}

// voiceOrder lists main first, then the cast keys in order.
func voiceOrder(key string) string {
	if key == common.AUDIO_VOICE_MAIN {
		return ""
	}
	return key
}

func (s *VoiceListService) voiceOf(ctx context.Context, key string, rows map[string]CastVoice, texts []string) (AdminVoice, []string) {
	svc := s.cast.Voice(ctx, key)
	v := AdminVoice{Key: key, FromEnv: true, Texts: len(texts)}
	if svc.enabled() {
		v.Profile, v.Credit = svc.voices[0].Profile, svc.voices[0].Credit
	}
	if row, ok := rows[key]; ok {
		updated := row.UpdatedAt
		v.FromEnv, v.Engine, v.StyleID, v.SpeakerName, v.StyleName = false, row.Engine, row.StyleID, row.SpeakerName, row.StyleName
		v.ModelUUID, v.Credit, v.UpdatedAt = row.ModelUUID, row.Credit, &updated
	}
	keys := clipKeys(svc, texts)
	v.Clips = len(keys)
	return v, keys
}

// clipKeys are the stored-clip keys a voice needs for texts, one per distinct sentence.
func clipKeys(svc *SpeechService, texts []string) []string {
	if !svc.enabled() {
		return nil
	}
	seen := map[string]bool{}
	var keys []string
	for _, text := range texts {
		for _, sentence := range Sentences(text) {
			key := svc.objectKey(svc.voices[0], sentence)
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func countStored(ctx context.Context, cast *Cast, key string, texts []string, stored map[string]bool) int {
	n := 0
	for _, k := range clipKeys(cast.Voice(ctx, key), texts) {
		if stored[k] {
			n++
		}
	}
	return n
}

// VoiceSaver stores a key's voice; *Repository satisfies it.
type VoiceSaver interface {
	SaveCastVoice(ctx context.Context, v *CastVoice) error
}

// VoiceSetService gives a voice key another engine style. The new voice's
// clips are made by the worker or on demand; until then the key has none.
type VoiceSetService struct {
	admins SuperAdmins
	engine EngineAPI
	voices VoiceSaver
	cast   *Cast
	log    *zap.Logger
}

func NewVoiceSetService(admins SuperAdmins, engine EngineAPI, voices VoiceSaver, cast *Cast, log *zap.Logger) *VoiceSetService {
	return &VoiceSetService{admins: admins, engine: engine, voices: voices, cast: cast, log: log}
}

func (s *VoiceSetService) Set(ctx context.Context, userID uuid.UUID, key string, styleID int) (CastVoice, error) {
	if err := requireSuperAdmin(ctx, s.admins, userID); err != nil {
		return CastVoice{}, err
	}
	if key == common.AUDIO_VOICE_ALT {
		return CastVoice{}, exceptions.ErrAudioVoiceKeyUnknown
	}
	if !s.engine.Configured() {
		return CastVoice{}, exceptions.ErrAudioEngineUnavailable
	}
	speakers, err := s.engine.Speakers(ctx)
	if err != nil {
		s.log.Warn("audio admin: engine speakers", zap.Error(err))
		return CastVoice{}, exceptions.ErrAudioEngineUnavailable
	}
	speaker, style, ok := findStyle(speakers, styleID)
	if !ok {
		return CastVoice{}, exceptions.ErrAudioStyleNotFound
	}
	model, err := s.engine.ModelOf(ctx, speaker.SpeakerUUID)
	if err != nil {
		s.log.Warn("audio admin: engine models", zap.Error(err))
		return CastVoice{}, exceptions.ErrAudioEngineUnavailable
	}
	v := CastVoice{
		Key: key, Engine: s.engine.Name(), StyleID: style.ID, SpeakerName: speaker.Name, StyleName: style.Name,
		ModelUUID: model, Credit: creditFor(s.engine.Name(), speaker.Name), UpdatedBy: &userID, UpdatedAt: time.Now(),
	}
	if err := s.voices.SaveCastVoice(ctx, &v); err != nil {
		return CastVoice{}, fmt.Errorf("audio admin: %w", err)
	}
	s.cast.Invalidate()
	return v, nil
}

func findStyle(speakers []EngineSpeaker, styleID int) (EngineSpeaker, EngineStyle, bool) {
	for _, sp := range speakers {
		for _, st := range sp.Styles {
			if st.ID == styleID {
				return sp, st, true
			}
		}
	}
	return EngineSpeaker{}, EngineStyle{}, false
}

func creditFor(engine, speaker string) string {
	name, ok := common.AUDIO_ENGINE_CREDIT_NAMES[engine]
	if !ok {
		name = engine
	}
	return name + ":" + speaker
}

// EngineStatusService reports whether the engine answers, and its voices.
type EngineStatusService struct {
	admins SuperAdmins
	engine EngineAPI
	log    *zap.Logger
}

func NewEngineStatusService(admins SuperAdmins, engine EngineAPI, log *zap.Logger) *EngineStatusService {
	return &EngineStatusService{admins: admins, engine: engine, log: log}
}

func (s *EngineStatusService) Status(ctx context.Context, userID uuid.UUID) (EngineStatus, error) {
	if err := requireSuperAdmin(ctx, s.admins, userID); err != nil {
		return EngineStatus{}, err
	}
	status := EngineStatus{Configured: s.engine.Configured(), Engine: s.engine.Name(), Speakers: []EngineSpeaker{}}
	if !status.Configured {
		return status, nil
	}
	version, err := s.engine.Version(ctx)
	if err != nil {
		s.log.Warn("audio admin: engine version", zap.Error(err))
		return status, nil
	}
	speakers, err := s.engine.Speakers(ctx)
	if err != nil {
		s.log.Warn("audio admin: engine speakers", zap.Error(err))
		return status, nil
	}
	status.Reachable, status.Version, status.Speakers = true, version, speakers
	return status, nil
}

// PreviewService speaks a text in any style for the admin to hear; the clip
// is returned, never stored.
type PreviewService struct {
	admins SuperAdmins
	engine EngineAPI
	log    *zap.Logger
}

func NewPreviewService(admins SuperAdmins, engine EngineAPI, log *zap.Logger) *PreviewService {
	return &PreviewService{admins: admins, engine: engine, log: log}
}

func (s *PreviewService) Preview(ctx context.Context, userID uuid.UUID, req PreviewRequest) (PreviewResponse, error) {
	if err := requireSuperAdmin(ctx, s.admins, userID); err != nil {
		return PreviewResponse{}, err
	}
	if !s.engine.Configured() {
		return PreviewResponse{}, exceptions.ErrAudioEngineUnavailable
	}
	wav, err := s.engine.Synth(req.StyleID).Synthesize(ctx, req.Text)
	if err != nil {
		s.log.Warn("audio admin: preview", zap.Int("style", req.StyleID), zap.Error(err))
		return PreviewResponse{}, exceptions.ErrAudioEngineUnavailable
	}
	return PreviewResponse{ContentType: WAVPassthrough{}.ContentType(), Audio: normalizePeak(wav)}, nil
}

// JobAdminStore queues and lists worker jobs; *Repository satisfies it.
type JobAdminStore interface {
	CreateJob(ctx context.Context, j *Job) error
	ListJobs(ctx context.Context, limit int) ([]Job, error)
	CancelJob(ctx context.Context, id uuid.UUID) (bool, error)
}

// JobService lets a super-admin queue work for the audio worker.
type JobService struct {
	admins SuperAdmins
	jobs   JobAdminStore
	log    *zap.Logger
}

func NewJobService(admins SuperAdmins, jobs JobAdminStore, log *zap.Logger) *JobService {
	return &JobService{admins: admins, jobs: jobs, log: log}
}

func (s *JobService) Create(ctx context.Context, userID uuid.UUID, req CreateJobRequest) (JobDTO, error) {
	if err := requireSuperAdmin(ctx, s.admins, userID); err != nil {
		return JobDTO{}, err
	}
	j := Job{ID: uuid.New(), Kind: req.Kind, Status: common.AudioJobQueued, CreatedBy: &userID, CreatedAt: time.Now()}
	switch req.Kind {
	case common.AudioJobInstallModel:
		if req.ModelUUID == nil {
			return JobDTO{}, exceptions.ErrAudioJobInvalid
		}
		j.ModelUUID = req.ModelUUID
	case common.AudioJobGenerate:
		if req.VoiceKey == common.AUDIO_VOICE_ALT {
			return JobDTO{}, exceptions.ErrAudioVoiceKeyUnknown
		}
		if req.VoiceKey != "" {
			j.VoiceKey = &req.VoiceKey
		}
	default:
		return JobDTO{}, exceptions.ErrAudioJobInvalid
	}
	if err := s.jobs.CreateJob(ctx, &j); err != nil {
		return JobDTO{}, fmt.Errorf("audio admin: %w", err)
	}
	return jobDTO(&j, time.Now()), nil
}

func (s *JobService) List(ctx context.Context, userID uuid.UUID) (JobsResponse, error) {
	if err := requireSuperAdmin(ctx, s.admins, userID); err != nil {
		return JobsResponse{}, err
	}
	jobs, err := s.jobs.ListJobs(ctx, common.AUDIO_JOBS_LIST_LIMIT)
	if err != nil {
		return JobsResponse{}, fmt.Errorf("audio admin: %w", err)
	}
	now := time.Now()
	out := JobsResponse{Jobs: make([]JobDTO, len(jobs))}
	for i := range jobs {
		out.Jobs[i] = jobDTO(&jobs[i], now)
	}
	return out, nil
}

func (s *JobService) Cancel(ctx context.Context, userID, id uuid.UUID) error {
	if err := requireSuperAdmin(ctx, s.admins, userID); err != nil {
		return err
	}
	ok, err := s.jobs.CancelJob(ctx, id)
	if err != nil {
		return fmt.Errorf("audio admin: %w", err)
	}
	if !ok {
		return exceptions.ErrAudioJobNotFound
	}
	return nil
}
