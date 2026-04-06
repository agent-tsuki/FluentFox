// Package xp — model.go.
// DB models and response DTOs for the XP and levelling domain.
package xp

import (
	"time"

	"github.com/google/uuid"
)

// UserXP is the DB model for the user_xp table.
type UserXP struct {
	UserID    uuid.UUID `db:"user_id"`
	Total     int       `db:"total_xp"`
	Level     int       `db:"level"`
	UpdatedAt time.Time `db:"updated_at"`
}

// XPTransaction is the DB model for the xp_transactions table.
type XPTransaction struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	Amount     int       `db:"amount"`
	Source     string    `db:"source"`      // srs_review, quiz_correct, chapter_complete, etc.
	SourceID   *uuid.UUID `db:"source_id"`   // optional reference to the triggering entity
	CreatedAt  time.Time  `db:"created_at"`
}

// XPLevelConfig is the DB model for the xp_level_config table.
type XPLevelConfig struct {
	Level        int `db:"level"`
	XPRequired   int `db:"xp_required"` // total XP needed to reach this level
}

// XPRewardConfig is the DB model for the xp_reward_config table.
type XPRewardConfig struct {
	Source  string `db:"source"`
	Amount  int    `db:"amount"`
}

// --- Response DTOs ---

// XPResponse is the public representation of a user's XP and level.
type XPResponse struct {
	Total       int     `json:"total_xp"`
	Level       int     `json:"level"`
	XPToNextLevel int   `json:"xp_to_next_level"`
	Progress    float64 `json:"progress_percent"`
}

// LeaderboardEntry is a single row in the XP leaderboard.
type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Total    int    `json:"total_xp"`
	Level    int    `json:"level"`
}
