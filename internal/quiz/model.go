// Package quiz — model.go.
package quiz

import (
	"time"

	"github.com/google/uuid"
)

// QuizSession is the DB model for the quiz_sessions table.
type QuizSession struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id"`
	QuizType    string     `db:"quiz_type"`   // multiple_choice, fill_blank, listening
	ChapterID   *uuid.UUID `db:"chapter_id"`
	TotalCards  int        `db:"total_cards"`
	Correct     int        `db:"correct"`
	Incorrect   int        `db:"incorrect"`
	CompletedAt *time.Time `db:"completed_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

// QuizAnswer is the DB model for the quiz_answers table.
type QuizAnswer struct {
	ID          uuid.UUID `db:"id"`
	SessionID   uuid.UUID `db:"session_id"`
	ContentID   uuid.UUID `db:"content_id"`
	UserAnswer  string    `db:"user_answer"`
	Correct     bool      `db:"correct"`
	TimeTakenMs int       `db:"time_taken_ms"`
}

// --- Request DTOs ---

// StartSessionRequest is the payload for POST /quiz/sessions.
type StartSessionRequest struct {
	QuizType  string  `json:"quiz_type"   validate:"required,oneof=multiple_choice fill_blank listening"`
	ChapterID *string `json:"chapter_id"  validate:"omitempty,uuid"`
	CardCount int     `json:"card_count"  validate:"omitempty,min=5,max=50"`
}

// SubmitAnswerRequest is the payload for POST /quiz/sessions/{id}/answers.
type SubmitAnswerRequest struct {
	ContentID   string `json:"content_id"   validate:"required,uuid"`
	UserAnswer  string `json:"user_answer"  validate:"required"`
	TimeTakenMs int    `json:"time_taken_ms" validate:"required,min=0"`
}

// --- Response DTOs ---

// SessionResponse is returned when a quiz session is created or retrieved.
type SessionResponse struct {
	ID        string  `json:"id"`
	QuizType  string  `json:"quiz_type"`
	ChapterID *string `json:"chapter_id"`
	TotalCards int    `json:"total_cards"`
}

// AnswerResultResponse is returned after submitting an answer.
type AnswerResultResponse struct {
	Correct       bool   `json:"correct"`
	CorrectAnswer string `json:"correct_answer"`
	Explanation   string `json:"explanation,omitempty"`
	XPEarned      int    `json:"xp_earned"`
}

// SessionSummaryResponse is returned when a session is completed.
type SessionSummaryResponse struct {
	SessionID string  `json:"session_id"`
	Correct   int     `json:"correct"`
	Incorrect int     `json:"incorrect"`
	Accuracy  float64 `json:"accuracy"`
	XPEarned  int     `json:"xp_earned"`
}
