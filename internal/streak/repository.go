// Package streak — repository.go.
// Owns all SQL for user_streaks and streak_activity_log.
package streak

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles streak-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a streak Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetByUserID fetches the streak record for a user.
func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserStreak, error) {
	s := &UserStreak{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, current_streak, longest_streak, last_activity_date, freeze_count, updated_at
		 FROM user_streaks WHERE user_id = $1`, userID,
	).Scan(&s.UserID, &s.CurrentStreak, &s.LongestStreak, &s.LastActivityDate, &s.FreezeCount, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("streak repository: get by user id: %w", err)
	}
	return s, nil
}

// UpsertStreak inserts or updates the streak record.
func (r *Repository) UpsertStreak(ctx context.Context, s *UserStreak) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_streaks (user_id, current_streak, longest_streak, last_activity_date, freeze_count, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (user_id) DO UPDATE
		 SET current_streak = EXCLUDED.current_streak,
		     longest_streak = EXCLUDED.longest_streak,
		     last_activity_date = EXCLUDED.last_activity_date,
		     freeze_count = EXCLUDED.freeze_count,
		     updated_at = NOW()`,
		s.UserID, s.CurrentStreak, s.LongestStreak, s.LastActivityDate, s.FreezeCount,
	)
	if err != nil {
		return fmt.Errorf("streak repository: upsert streak: %w", err)
	}
	return nil
}

// LogActivity records a daily activity event.
func (r *Repository) LogActivity(ctx context.Context, userID uuid.UUID, activityType string) error {
	today := time.Now().Truncate(24 * time.Hour)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO streak_activity_log (id, user_id, activity_date, activity_type)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, activity_date) DO NOTHING`,
		uuid.New(), userID, today, activityType,
	)
	if err != nil {
		return fmt.Errorf("streak repository: log activity: %w", err)
	}
	return nil
}
