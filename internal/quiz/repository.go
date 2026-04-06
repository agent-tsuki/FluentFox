// Package quiz — repository.go.
// Owns all SQL for quiz sessions and answers.
package quiz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles quiz-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a quiz Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateSession inserts a new quiz session and returns it.
func (r *Repository) CreateSession(ctx context.Context, userID uuid.UUID, quizType string, chapterID *uuid.UUID, totalCards int) (*QuizSession, error) {
	s := &QuizSession{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO quiz_sessions (id, user_id, quiz_type, chapter_id, total_cards, correct, incorrect)
		 VALUES ($1, $2, $3, $4, $5, 0, 0)
		 RETURNING id, user_id, quiz_type, chapter_id, total_cards, correct, incorrect, completed_at, created_at`,
		uuid.New(), userID, quizType, chapterID, totalCards,
	).Scan(&s.ID, &s.UserID, &s.QuizType, &s.ChapterID, &s.TotalCards,
		&s.Correct, &s.Incorrect, &s.CompletedAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("quiz repository: create session: %w", err)
	}
	return s, nil
}

// GetSession fetches a quiz session by its ID.
func (r *Repository) GetSession(ctx context.Context, id uuid.UUID) (*QuizSession, error) {
	s := &QuizSession{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, quiz_type, chapter_id, total_cards, correct, incorrect, completed_at, created_at
		 FROM quiz_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.UserID, &s.QuizType, &s.ChapterID, &s.TotalCards,
		&s.Correct, &s.Incorrect, &s.CompletedAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("quiz repository: get session: %w", err)
	}
	return s, nil
}

// RecordAnswer inserts a quiz answer and increments the session correct/incorrect counter.
func (r *Repository) RecordAnswer(ctx context.Context, sessionID, contentID uuid.UUID, userAnswer string, correct bool, timeTakenMs int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO quiz_answers (id, session_id, content_id, user_answer, correct, time_taken_ms)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), sessionID, contentID, userAnswer, correct, timeTakenMs,
	)
	if err != nil {
		return fmt.Errorf("quiz repository: record answer: %w", err)
	}

	column := "incorrect"
	if correct {
		column = "correct"
	}
	_, err = r.pool.Exec(ctx,
		fmt.Sprintf(`UPDATE quiz_sessions SET %s = %s + 1 WHERE id = $1`, column, column),
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("quiz repository: update session counters: %w", err)
	}
	return nil
}

// CompleteSession sets completed_at for the session.
func (r *Repository) CompleteSession(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE quiz_sessions SET completed_at = $1 WHERE id = $2`, now, id)
	if err != nil {
		return fmt.Errorf("quiz repository: complete session: %w", err)
	}
	return nil
}
