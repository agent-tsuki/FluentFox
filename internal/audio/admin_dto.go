package audio

import (
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

type AdminVoice struct {
	Key         string     `json:"key"                    doc:"main, or a listening cast key such as f1"`
	FromEnv     bool       `json:"from_env"               doc:"No cast row: the server's env voice speaks this key (main) or the main voice does"`
	Profile     string     `json:"profile"                doc:"Clip folder, e.g. aivisspeech-888753760"`
	Engine      string     `json:"engine,omitempty"`
	StyleID     int        `json:"style_id,omitempty"`
	SpeakerName string     `json:"speaker_name,omitempty"`
	StyleName   string     `json:"style_name,omitempty"`
	ModelUUID   *uuid.UUID `json:"model_uuid,omitempty"   doc:"AivisHub model the style comes from"`
	Credit      string     `json:"credit,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	Texts       int        `json:"texts"                  doc:"Course texts this voice speaks"`
	Clips       int        `json:"clips"                  doc:"Clips those texts need (one per sentence)"`
	Ready       int        `json:"ready"                  doc:"Of those clips, how many are stored"`
}

type AdminVoicesResponse struct {
	Voices []AdminVoice `json:"voices"`
}

type EngineStatus struct {
	Configured bool            `json:"configured" doc:"VOICEVOX_URL is set"`
	Reachable  bool            `json:"reachable"`
	Engine     string          `json:"engine"     doc:"Engine name in clip folders, e.g. aivisspeech"`
	Version    string          `json:"version,omitempty"`
	Speakers   []EngineSpeaker `json:"speakers"   doc:"Voices the engine has, with their styles"`
}

type VoiceKeyPath struct {
	Key string `path:"key" pattern:"^[a-z][a-z0-9]{0,15}$" doc:"main or a cast key"`
}

type SetVoiceInput struct {
	VoiceKeyPath
	Body SetVoiceRequest
}

type SetVoiceRequest struct {
	StyleID int `json:"style_id" minimum:"0" doc:"Engine style id from GET /admin/audio/engine"`
}

type PreviewRequest struct {
	StyleID int    `json:"style_id" minimum:"0"`
	Text    string `json:"text"     minLength:"1" maxLength:"120" doc:"Japanese text to hear in this style; not stored"`
}

type PreviewResponse struct {
	ContentType string `json:"content_type"`
	Audio       []byte `json:"audio" doc:"The clip, base64"`
}

type CreateJobRequest struct {
	Kind      common.AudioJobKind `json:"kind"                 enum:"generate,install_model"`
	VoiceKey  string              `json:"voice_key,omitempty"  pattern:"^[a-z][a-z0-9]{0,15}$" doc:"generate: one voice; empty for every voice"`
	ModelUUID *uuid.UUID          `json:"model_uuid,omitempty" doc:"install_model: the AivisHub model"`
}

type JobIDPath struct {
	ID uuid.UUID `path:"id" format:"uuid"`
}

type JobDTO struct {
	ID         uuid.UUID             `json:"id"`
	Kind       common.AudioJobKind   `json:"kind"`
	Status     common.AudioJobStatus `json:"status"`
	VoiceKey   *string               `json:"voice_key,omitempty"`
	ModelUUID  *uuid.UUID            `json:"model_uuid,omitempty"`
	Total      int                   `json:"total"`
	Done       int                   `json:"done"`
	Failed     int                   `json:"failed"`
	Error      *string               `json:"error,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
	StartedAt  *time.Time            `json:"started_at,omitempty"`
	FinishedAt *time.Time            `json:"finished_at,omitempty"`
	// Alive is false for a running job its worker stopped marking.
	Alive bool `json:"alive"`
}

type JobsResponse struct {
	Jobs []JobDTO `json:"jobs"`
}

func jobDTO(j *Job, now time.Time) JobDTO {
	alive := j.Status != common.AudioJobRunning ||
		(j.HeartbeatAt != nil && now.Sub(*j.HeartbeatAt) < common.AUDIO_JOB_STALE_AFTER)
	return JobDTO{
		ID: j.ID, Kind: j.Kind, Status: j.Status, VoiceKey: j.VoiceKey, ModelUUID: j.ModelUUID,
		Total: j.Total, Done: j.Done, Failed: j.Failed, Error: j.Error,
		CreatedAt: j.CreatedAt, StartedAt: j.StartedAt, FinishedAt: j.FinishedAt, Alive: alive,
	}
}
