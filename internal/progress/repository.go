// Package progress — repository.go.
// Owns all SQL for user_chapter_progress and user_vocab_mastery.
package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles progress-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a progress Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetChapterProgress fetches the progress row for a user+chapter pair.
func (r *Repository) GetChapterProgress(ctx context.Context, userID, chapterID uuid.UUID) (*UserChapterProgress, error) {
	p := &UserChapterProgress{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, chapter_id, status, score, started_at, completed_at
		 FROM user_chapter_progress WHERE user_id = $1 AND chapter_id = $2`,
		userID, chapterID,
	).Scan(&p.UserID, &p.ChapterID, &p.Status, &p.Score, &p.StartedAt, &p.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("progress repository: get chapter progress: %w", err)
	}
	return p, nil
}

// UpsertChapterProgress inserts or updates progress for a chapter.
func (r *Repository) UpsertChapterProgress(ctx context.Context, userID, chapterID uuid.UUID, status string, score *int) error {
	now := time.Now()
	var completedAt *time.Time
	if status == "completed" {
		completedAt = &now
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_chapter_progress (user_id, chapter_id, status, score, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, NOW(), $5)
		 ON CONFLICT (user_id, chapter_id) DO UPDATE
		 SET status = EXCLUDED.status,
		     score = COALESCE(EXCLUDED.score, user_chapter_progress.score),
		     completed_at = COALESCE(EXCLUDED.completed_at, user_chapter_progress.completed_at)`,
		userID, chapterID, status, score, completedAt,
	)
	if err != nil {
		return fmt.Errorf("progress repository: upsert chapter progress: %w", err)
	}
	return nil
}

// GetOverallProgress returns aggregate stats for a user's progress.
func (r *Repository) GetOverallProgress(ctx context.Context, userID uuid.UUID) (chaptersCompleted, chaptersTotal, vocabMastered, vocabTotal int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT
		     (SELECT COUNT(*) FROM user_chapter_progress WHERE user_id = $1 AND status = 'completed'),
		     (SELECT COUNT(*) FROM chapters WHERE published = true),
		     (SELECT COUNT(*) FROM user_vocab_mastery WHERE user_id = $1 AND mastered = true),
		     (SELECT COUNT(*) FROM vocabulary)`,
		userID,
	).Scan(&chaptersCompleted, &chaptersTotal, &vocabMastered, &vocabTotal)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("progress repository: get overall progress: %w", err)
	}
	return chaptersCompleted, chaptersTotal, vocabMastered, vocabTotal, nil
}

// ListChapterProgress returns all progress rows for a user.
func (r *Repository) ListChapterProgress(ctx context.Context, userID uuid.UUID) ([]*UserChapterProgress, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, chapter_id, status, score, started_at, completed_at
		 FROM user_chapter_progress WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("progress repository: list chapter progress: %w", err)
	}
	defer rows.Close()

	var result []*UserChapterProgress
	for rows.Next() {
		p := &UserChapterProgress{}
		if err := rows.Scan(&p.UserID, &p.ChapterID, &p.Status, &p.Score, &p.StartedAt, &p.CompletedAt); err != nil {
			return nil, fmt.Errorf("progress repository: scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
