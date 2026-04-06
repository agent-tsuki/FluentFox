// Package progress — model.go.
// DB models and response DTOs for tracking user progress through content.
package progress

import (
	"time"

	"github.com/google/uuid"
)

// UserChapterProgress is the DB model for the user_chapter_progress table.
type UserChapterProgress struct {
	UserID     uuid.UUID  `db:"user_id"`
	ChapterID  uuid.UUID  `db:"chapter_id"`
	Status     string     `db:"status"` // not_started, in_progress, completed
	Score      *int       `db:"score"`
	StartedAt  *time.Time `db:"started_at"`
	CompletedAt *time.Time `db:"completed_at"`
}

// UserVocabMastery is the DB model for the user_vocab_mastery table.
type UserVocabMastery struct {
	UserID    uuid.UUID `db:"user_id"`
	VocabID   uuid.UUID `db:"vocab_id"`
	Mastered  bool      `db:"mastered"`
	UpdatedAt time.Time `db:"updated_at"`
}

// --- Response DTOs ---

// ChapterProgressResponse is the public progress state for a chapter.
type ChapterProgressResponse struct {
	ChapterID   string  `json:"chapter_id"`
	Status      string  `json:"status"`
	Score       *int    `json:"score"`
	CompletedAt *string `json:"completed_at"`
}

// OverallProgressResponse summarises a user's progress across all content.
type OverallProgressResponse struct {
	ChaptersCompleted int     `json:"chapters_completed"`
	ChaptersTotal     int     `json:"chapters_total"`
	VocabMastered     int     `json:"vocab_mastered"`
	VocabTotal        int     `json:"vocab_total"`
	CompletionPercent float64 `json:"completion_percent"`
}
