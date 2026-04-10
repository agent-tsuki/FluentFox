package vocabulary

import (
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Vocabulary struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Word        string           `gorm:"column:word;not null"`
	Meaning     string           `gorm:"column:meaning;not null"`
	Hiragana    *string          `gorm:"column:hiragana"`
	Romaji      *string          `gorm:"column:romaji"`
	TargetLevel common.JLPTLevel `gorm:"column:target_level;type:jlpt_level;not null"`
	ImageKey    *string          `gorm:"column:image_key"`
	AudioKey    *string          `gorm:"column:audio_key"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Vocabulary) TableName() string { return "vocabulary" }

func (v *Vocabulary) BeforeCreate(_ *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// KanjiVocabulary is the join table linking kanji to vocabulary entries.
type KanjiVocabulary struct {
	KanjiID      uuid.UUID `gorm:"column:kanji_id;primaryKey"`
	VocabularyID uuid.UUID `gorm:"column:vocabulary_id;primaryKey"`
}

func (KanjiVocabulary) TableName() string { return "kanji_vocabulary" }
