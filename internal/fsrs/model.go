package fsrs

import (
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FSRSCard holds the FSRS algorithm state for a single review card.
// content_type + content_id form a polymorphic reference to the content being
// reviewed (kanji, kana, vocabulary, etc.).
type FSRSCard struct {
	ID            uuid.UUID           `gorm:"type:uuid;primaryKey"`
	UserID        uuid.UUID           `gorm:"column:user_id;not null"`
	Due           time.Time           `gorm:"column:due;not null"`
	Stability     float64             `gorm:"column:stability;not null"`
	Difficulty    float64             `gorm:"column:difficulty;not null"`
	ElapsedDays   int                 `gorm:"column:elapsed_days;not null"`
	ScheduledDays int                 `gorm:"column:scheduled_days;not null"`
	Reps          int                 `gorm:"column:reps;not null;default:0"`
	Lapses        int                 `gorm:"column:lapses;not null;default:0"`
	State         string              `gorm:"column:state;not null"`
	LastReview    *time.Time          `gorm:"column:last_review"`
	ContentType   common.SRSContentType `gorm:"column:content_type;type:srs_content_type;not null"`
	ContentID     uuid.UUID           `gorm:"column:content_id;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (FSRSCard) TableName() string { return "fsrs_card" }

func (f *FSRSCard) BeforeCreate(_ *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// ReviewLog records each individual review event for a card.
// rating must be 1–4 (enforced by a DB CHECK constraint).
type ReviewLog struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	CardID        uuid.UUID `gorm:"column:card_id;not null"`
	Rating        int       `gorm:"column:rating;not null"`
	ScheduledDays int       `gorm:"column:scheduled_days;not null"`
	ElapsedDays   int       `gorm:"column:elapsed_days;not null"`
	ReviewAt      time.Time `gorm:"column:review_at;autoCreateTime"`
	State         string    `gorm:"column:state;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ReviewLog) TableName() string { return "review_log" }

func (r *ReviewLog) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
