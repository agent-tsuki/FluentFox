package audio

import (
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

// CastVoice is the engine style a voice key speaks with (audio_voice).
type CastVoice struct {
	Key         string     `gorm:"column:key;primaryKey"`
	Engine      string     `gorm:"column:engine"`
	StyleID     int        `gorm:"column:style_id"`
	SpeakerName string     `gorm:"column:speaker_name"`
	StyleName   string     `gorm:"column:style_name"`
	ModelUUID   *uuid.UUID `gorm:"column:model_uuid"`
	Credit      string     `gorm:"column:credit"`
	UpdatedBy   *uuid.UUID `gorm:"column:updated_by"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (CastVoice) TableName() string { return "audio_voice" }

// Profile names the voice's clip folder, e.g. "aivisspeech-888753760".
func (v *CastVoice) Profile() string {
	return EngineProfile(v.Engine, v.StyleID)
}

// ClipRecord is one stored clip (audio_clip).
type ClipRecord struct {
	ObjectKey string     `gorm:"column:object_key;primaryKey"`
	Profile   string     `gorm:"column:profile"`
	Text      string     `gorm:"column:text"`
	Bytes     int        `gorm:"column:bytes"`
	CreatedBy *uuid.UUID `gorm:"column:created_by"`
}

func (ClipRecord) TableName() string { return "audio_clip" }

// Job is work queued for the audio worker (audio_job).
type Job struct {
	ID          uuid.UUID             `gorm:"column:id;primaryKey"`
	Kind        common.AudioJobKind   `gorm:"column:kind"`
	Status      common.AudioJobStatus `gorm:"column:status"`
	VoiceKey    *string               `gorm:"column:voice_key"`
	ModelUUID   *uuid.UUID            `gorm:"column:model_uuid"`
	Total       int                   `gorm:"column:total"`
	Done        int                   `gorm:"column:done"`
	Failed      int                   `gorm:"column:failed"`
	Error       *string               `gorm:"column:error"`
	CreatedBy   *uuid.UUID            `gorm:"column:created_by"`
	CreatedAt   time.Time             `gorm:"column:created_at"`
	StartedAt   *time.Time            `gorm:"column:started_at"`
	HeartbeatAt *time.Time            `gorm:"column:heartbeat_at"`
	FinishedAt  *time.Time            `gorm:"column:finished_at"`
}

func (Job) TableName() string { return "audio_job" }

// CourseText is a text the course speaks, with the voice key that says it.
type CourseText struct {
	Voice string
	Text  string
}
