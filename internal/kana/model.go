package kana

import (
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Kana struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Character   string           `gorm:"column:character;not null"`
	Romanji     string           `gorm:"column:romanji;not null"`
	KanaType    common.KanaType  `gorm:"column:kana_type;type:kana_type;not null"`
	TargetLevel common.JLPTLevel `gorm:"column:target_level;type:jlpt_level;not null"`
	StrokeOrder *int             `gorm:"column:stroke_order"`
	ImageKey    *string          `gorm:"column:image_key"`
	AudioKey    *string          `gorm:"column:audio_key"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Kana) TableName() string { return "kanas" }

func (k *Kana) BeforeCreate(_ *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}
