package quiz

import (
	"encoding/json"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuizSession struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey"`
	UserID         uuid.UUID        `gorm:"column:user_id;not null"`
	TargetLevel    common.JLPTLevel `gorm:"column:target_level;type:jlpt_level;not null"`
	TotalQuestions int              `gorm:"column:total_questions;not null;default:0"`
	CorrectCount   int              `gorm:"column:correct_count;not null;default:0"`
	CompletedAt    *time.Time       `gorm:"column:completed_at"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (QuizSession) TableName() string { return "quiz_sessions" }

func (q *QuizSession) BeforeCreate(_ *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

// QuizQuestion holds the question content. Options is a JSONB array of answer
// choices stored as raw JSON so the shape can evolve without schema changes.
type QuizQuestion struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey"`
	ContentType common.QuizType  `gorm:"column:content_type;type:quiz_type;not null"`
	ContentID   uuid.UUID        `gorm:"column:content_id;not null"`
	Question    string           `gorm:"column:question;not null"`
	Options     json.RawMessage  `gorm:"column:options;type:jsonb;not null"`
	CorrectID   string           `gorm:"column:correct_id;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (QuizQuestion) TableName() string { return "quiz_questions" }

func (q *QuizQuestion) BeforeCreate(_ *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

type QuizAnswer struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	QuestionID uuid.UUID `gorm:"column:question_id;not null"`
	SessionID  uuid.UUID `gorm:"column:session_id;not null"`
	Selected   string    `gorm:"column:selected;not null"`
	CorrectAns string    `gorm:"column:correct_ans;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (QuizAnswer) TableName() string { return "quiz_answers" }

func (q *QuizAnswer) BeforeCreate(_ *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}
