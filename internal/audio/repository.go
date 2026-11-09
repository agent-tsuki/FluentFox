package audio

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/fluentfox/api/internal/common"
)

// jobFinishedAt is set when a job stops, whichever way.
const jobFinishedAt = "finished_at"

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListCastVoices(ctx context.Context) ([]CastVoice, error) {
	var voices []CastVoice
	if err := r.db.WithContext(ctx).Order("key").Find(&voices).Error; err != nil {
		return nil, fmt.Errorf("list cast voices: %w", err)
	}
	return voices, nil
}

func (r *Repository) SaveCastVoice(ctx context.Context, v *CastVoice) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"engine": v.Engine, "style_id": v.StyleID, "speaker_name": v.SpeakerName, "style_name": v.StyleName,
			"model_uuid": v.ModelUUID, "credit": v.Credit, "updated_by": v.UpdatedBy, "updated_at": gorm.Expr("now()"),
		}),
	}).Create(v).Error
	if err != nil {
		return fmt.Errorf("save cast voice %s: %w", v.Key, err)
	}
	return nil
}

// ExistingClips reports which of the object keys have a stored clip.
func (r *Repository) ExistingClips(ctx context.Context, keys []string) (map[string]bool, error) {
	found := make(map[string]bool, len(keys))
	for start := 0; start < len(keys); start += common.AUDIO_CLIP_LOOKUP_MAX_KEYS {
		end := min(start+common.AUDIO_CLIP_LOOKUP_MAX_KEYS, len(keys))
		var have []string
		err := r.db.WithContext(ctx).Model(&ClipRecord{}).Where("object_key IN ?", keys[start:end]).Pluck("object_key", &have).Error
		if err != nil {
			return nil, fmt.Errorf("look up clips: %w", err)
		}
		for _, k := range have {
			found[k] = true
		}
	}
	return found, nil
}

// RecordClip adds a stored clip; one already recorded is left as it is.
func (r *Repository) RecordClip(ctx context.Context, c *ClipRecord) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(c).Error; err != nil {
		return fmt.Errorf("record clip %s: %w", c.ObjectKey, err)
	}
	return nil
}

// Listening text is written with readings ("私[わたし] は 学生[がくせい] です。");
// what is spoken has neither the readings nor the spaces, exactly as clients send it.
var (
	readingMarkup = regexp.MustCompile(`\[[^\]]*\]`)
	whitespace    = regexp.MustCompile(`\s+`)
)

func spokenText(markup string) string {
	return whitespace.ReplaceAllString(readingMarkup.ReplaceAllString(markup, ""), "")
}

// CourseTexts lists what the course speaks: each listening line in its
// speaker's voice key (a monologue's reader is speaker 0; a passage with no
// voices uses main), and each chapter word's kana in the main voice.
func (r *Repository) CourseTexts(ctx context.Context) ([]CourseText, error) {
	var lines []struct {
		Voice *string
		Text  string
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT p.speaker_voices[COALESCE(l.speaker, 0) + 1] AS voice, l.text
		FROM chapter_passage_line l JOIN chapter_passage p ON p.id = l.passage_id
		ORDER BY p.chapter_id, p.position, l.position`).Scan(&lines).Error
	if err != nil {
		return nil, fmt.Errorf("list listening lines: %w", err)
	}
	var words []string
	if err := r.db.WithContext(ctx).Table("chapter_word").Order("chapter_id, position").Pluck("kana", &words).Error; err != nil {
		return nil, fmt.Errorf("list chapter words: %w", err)
	}
	texts := make([]CourseText, 0, len(lines)+len(words))
	for _, l := range lines {
		voice := common.AUDIO_VOICE_MAIN
		if l.Voice != nil && *l.Voice != "" {
			voice = *l.Voice
		}
		texts = append(texts, CourseText{Voice: voice, Text: spokenText(l.Text)})
	}
	for _, w := range words {
		texts = append(texts, CourseText{Voice: common.AUDIO_VOICE_MAIN, Text: w})
	}
	return texts, nil
}

func (r *Repository) CreateJob(ctx context.Context, j *Job) error {
	if err := r.db.WithContext(ctx).Create(j).Error; err != nil {
		return fmt.Errorf("create audio job: %w", err)
	}
	return nil
}

func (r *Repository) ListJobs(ctx context.Context, limit int) ([]Job, error) {
	var jobs []Job
	if err := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("list audio jobs: %w", err)
	}
	return jobs, nil
}

// CancelJob stops a queued or running job; the worker sees it at its next batch.
func (r *Repository) CancelJob(ctx context.Context, id uuid.UUID) (bool, error) {
	res := r.db.WithContext(ctx).Model(&Job{}).
		Where("id = ? AND status IN ?", id, []common.AudioJobStatus{common.AudioJobQueued, common.AudioJobRunning}).
		Updates(map[string]any{"status": common.AudioJobCancelled, jobFinishedAt: gorm.Expr("now()")})
	if res.Error != nil {
		return false, fmt.Errorf("cancel audio job %s: %w", id, res.Error)
	}
	return res.RowsAffected > 0, nil
}

// RequeueStaleJobs puts back jobs whose worker stopped marking them.
func (r *Repository) RequeueStaleJobs(ctx context.Context, staleAfter time.Duration) error {
	err := r.db.WithContext(ctx).Model(&Job{}).
		Where("status = ? AND heartbeat_at < ?", common.AudioJobRunning, time.Now().Add(-staleAfter)).
		Update("status", common.AudioJobQueued).Error
	if err != nil {
		return fmt.Errorf("requeue stale audio jobs: %w", err)
	}
	return nil
}

// ClaimJob takes the oldest queued job; SKIP LOCKED lets two workers run
// without taking the same one. Nil when the queue is empty.
func (r *Repository) ClaimJob(ctx context.Context) (*Job, error) {
	var jobs []Job
	err := r.db.WithContext(ctx).Raw(`
		UPDATE audio_job SET status = ?, started_at = COALESCE(started_at, now()), heartbeat_at = now()
		WHERE id = (
			SELECT id FROM audio_job WHERE status = ? ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1
		)
		RETURNING *`, common.AudioJobRunning, common.AudioJobQueued).Scan(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("claim audio job: %w", err)
	}
	if len(jobs) == 0 {
		return nil, nil
	}
	return &jobs[0], nil
}

// errJobStopped means the job is no longer running (cancelled, or requeued
// after this worker was thought dead), so the worker drops it.
var errJobStopped = errors.New("audio: job no longer running")

// SaveJobProgress records progress and marks the job alive.
func (r *Repository) SaveJobProgress(ctx context.Context, j *Job) error {
	res := r.db.WithContext(ctx).Model(&Job{}).Where("id = ? AND status = ?", j.ID, common.AudioJobRunning).
		Updates(jobCounts(j, "heartbeat_at"))
	if res.Error != nil {
		return fmt.Errorf("save audio job %s progress: %w", j.ID, res.Error)
	}
	if res.RowsAffected == 0 {
		return errJobStopped
	}
	return nil
}

// FinishJob ends a running job; a job cancelled meanwhile stays cancelled.
func (r *Repository) FinishJob(ctx context.Context, j *Job, status common.AudioJobStatus, jobErr string) error {
	updates := jobCounts(j, jobFinishedAt)
	updates["status"] = status
	if jobErr != "" {
		updates["error"] = jobErr
	}
	err := r.db.WithContext(ctx).Model(&Job{}).Where("id = ? AND status = ?", j.ID, common.AudioJobRunning).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("finish audio job %s: %w", j.ID, err)
	}
	return nil
}

// jobCounts is a job's progress, with the time column to set to now.
func jobCounts(j *Job, nowColumn string) map[string]any {
	return map[string]any{"total": j.Total, "done": j.Done, "failed": j.Failed, nowColumn: gorm.Expr("now()")}
}

// CastModels lists the AivisHub models the cast's voices come from.
func (r *Repository) CastModels(ctx context.Context) ([]string, error) {
	var models []string
	err := r.db.WithContext(ctx).Raw(`SELECT DISTINCT model_uuid::text FROM audio_voice WHERE model_uuid IS NOT NULL`).Scan(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list cast models: %w", err)
	}
	return models, nil
}
