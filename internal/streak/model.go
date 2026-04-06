// Package streak — model.go.
package streak

import (
	"time"

	"github.com/google/uuid"
)

// UserStreak is the DB model for the user_streaks table.
type UserStreak struct {
	UserID          uuid.UUID  `db:"user_id"`
	CurrentStreak   int        `db:"current_streak"`
	LongestStreak   int        `db:"longest_streak"`
	LastActivityDate *time.Time `db:"last_activity_date"`
	FreezeCount     int        `db:"freeze_count"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

// StreakActivityLog is the DB model for the streak_activity_log table.
type StreakActivityLog struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	ActivityDate time.Time `db:"activity_date"`
	ActivityType string    `db:"activity_type"`
}

// StreakResponse is the public representation of a user's streak.
type StreakResponse struct {
	CurrentStreak   int        `json:"current_streak"`
	LongestStreak   int        `json:"longest_streak"`
	LastActivityDate *string   `json:"last_activity_date"`
	FreezeCount     int        `json:"freeze_count"`
	IsAlive         bool       `json:"is_alive"`
}
