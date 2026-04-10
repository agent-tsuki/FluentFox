package kanji

import (
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Kanji struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Word        string           `gorm:"column:word;not null"`
	Onyomi      *string          `gorm:"column:onyomi"`
	Kunyomi     *string          `gorm:"column:kunyomi"`
	Meaning     string           `gorm:"column:meaning;not null"`
	Hiragana    *string          `gorm:"column:hiragana"`
	Romaji      *string          `gorm:"column:romaji"`
	TargetLevel common.JLPTLevel `gorm:"column:target_level;type:jlpt_level;not null"`
	ImageKey    *string          `gorm:"column:image_key"`
	AudioKey    *string          `gorm:"column:audio_key"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Kanji) TableName() string { return "kanji" }

func (k *Kanji) BeforeCreate(_ *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}
