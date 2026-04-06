// Package xp — repository.go.
// Owns all SQL for user_xp, xp_transactions, xp_level_config, xp_reward_config.
package xp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles XP-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs an XP Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetByUserID fetches the XP record for a user.
func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserXP, error) {
	x := &UserXP{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, total_xp, level, updated_at FROM user_xp WHERE user_id = $1`, userID,
	).Scan(&x.UserID, &x.Total, &x.Level, &x.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("xp repository: get by user id: %w", err)
	}
	return x, nil
}

// AddXP increments the user's total XP and records the transaction.
// Returns the updated total.
func (r *Repository) AddXP(ctx context.Context, userID uuid.UUID, amount int, source string, sourceID *uuid.UUID) (int, error) {
	var newTotal int
	err := r.pool.QueryRow(ctx,
		`INSERT INTO user_xp (user_id, total_xp, level, updated_at)
		 VALUES ($1, $2, 1, NOW())
		 ON CONFLICT (user_id) DO UPDATE
		 SET total_xp = user_xp.total_xp + $2, updated_at = NOW()
		 RETURNING total_xp`,
		userID, amount,
	).Scan(&newTotal)
	if err != nil {
		return 0, fmt.Errorf("xp repository: add xp: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO xp_transactions (id, user_id, amount, source, source_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		uuid.New(), userID, amount, source, sourceID,
	)
	if err != nil {
		return 0, fmt.Errorf("xp repository: record transaction: %w", err)
	}

	return newTotal, nil
}

// GetLevelConfig returns the XP thresholds for all levels.
func (r *Repository) GetLevelConfig(ctx context.Context) ([]*XPLevelConfig, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT level, xp_required FROM xp_level_config ORDER BY level`)
	if err != nil {
		return nil, fmt.Errorf("xp repository: get level config: %w", err)
	}
	defer rows.Close()

	var configs []*XPLevelConfig
	for rows.Next() {
		c := &XPLevelConfig{}
		if err := rows.Scan(&c.Level, &c.XPRequired); err != nil {
			return nil, fmt.Errorf("xp repository: scan level config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpdateLevel sets the user's computed level.
func (r *Repository) UpdateLevel(ctx context.Context, userID uuid.UUID, level int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE user_xp SET level = $1, updated_at = NOW() WHERE user_id = $2`, level, userID)
	if err != nil {
		return fmt.Errorf("xp repository: update level: %w", err)
	}
	return nil
}

// GetLeaderboard returns the top N users by total XP.
func (r *Repository) GetLeaderboard(ctx context.Context, limit int) ([]*LeaderboardEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ROW_NUMBER() OVER (ORDER BY ux.total_xp DESC) AS rank,
		        ux.user_id, u.username, ux.total_xp, ux.level
		 FROM user_xp ux
		 JOIN users u ON u.id = ux.user_id
		 ORDER BY ux.total_xp DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("xp repository: get leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []*LeaderboardEntry
	for rows.Next() {
		e := &LeaderboardEntry{}
		var userID uuid.UUID
		if err := rows.Scan(&e.Rank, &userID, &e.Username, &e.Total, &e.Level); err != nil {
			return nil, fmt.Errorf("xp repository: scan leaderboard: %w", err)
		}
		e.UserID = userID.String()
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
