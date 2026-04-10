package grammar

import (
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Grammar struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey"`
	ChapterNo   int              `gorm:"column:chapter_no;not null"`
	Title       string           `gorm:"column:title;not null"`
	TargetLevel common.JLPTLevel `gorm:"column:target_level;type:jlpt_level;not null"`
	Content     string           `gorm:"column:content;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Grammar) TableName() string { return "grammar" }

func (g *Grammar) BeforeCreate(_ *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
