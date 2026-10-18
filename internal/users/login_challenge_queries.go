package users

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoginChallenge struct {
	ID         uuid.UUID        `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID        `gorm:"column:user_id;not null"`
	CodeHash   string           `gorm:"column:code_hash;not null"`
	Attempts   int              `gorm:"column:attempts;not null;default:0"`
	Location   *SessionLocation `gorm:"column:location;serializer:json"`
	ExpiresAt  time.Time        `gorm:"column:expires_at;not null"`
	ConsumedAt *time.Time       `gorm:"column:consumed_at"`
	CreatedAt  time.Time        `gorm:"column:created_at;autoCreateTime"`
}

func (LoginChallenge) TableName() string { return "login_challenges" }

type LoginChallengeStats struct {
	Count  int64
	Latest *time.Time
}

func (r *Repository) GetLoginChallengeStats(ctx context.Context, tx *gorm.DB, userID uuid.UUID, since time.Time) (LoginChallengeStats, error) {
	var stats LoginChallengeStats
	if err := tx.WithContext(ctx).Model(&LoginChallenge{}).
		Select("COUNT(*) AS count, MAX(created_at) AS latest").
		Where("user_id = ? AND created_at >= ?", userID, since).
		Scan(&stats).Error; err != nil {
		return LoginChallengeStats{}, fmt.Errorf("login challenge stats: %w", err)
	}
	return stats, nil
}

func (r *Repository) CreateLoginChallenge(ctx context.Context, tx *gorm.DB, c *LoginChallenge) error {
	if err := tx.WithContext(ctx).Create(c).Error; err != nil {
		return fmt.Errorf("create login challenge: %w", err)
	}
	return nil
}

// The row lock serialises parallel guesses so the attempt limit cannot be raced.
func (r *Repository) LockLoginChallenge(ctx context.Context, tx *gorm.DB, id uuid.UUID) (LoginChallenge, error) {
	var c LoginChallenge
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", id).
		First(&c).Error; err != nil {
		return LoginChallenge{}, fmt.Errorf("lock login challenge: %w", err)
	}
	return c, nil
}

func (r *Repository) RecordLoginChallengeAttempt(ctx context.Context, tx *gorm.DB, id uuid.UUID, attempts int) error {
	if err := tx.WithContext(ctx).Model(&LoginChallenge{}).Where("id = ?", id).
		Update("attempts", attempts).Error; err != nil {
		return fmt.Errorf("record login challenge attempt: %w", err)
	}
	return nil
}

func (r *Repository) ConsumeLoginChallenge(ctx context.Context, tx *gorm.DB, id uuid.UUID, now time.Time) error {
	if err := tx.WithContext(ctx).Model(&LoginChallenge{}).Where("id = ?", id).
		Update("consumed_at", now).Error; err != nil {
		return fmt.Errorf("consume login challenge: %w", err)
	}
	return nil
}

// Run inside each new challenge, since there is no background job: rows older
// than the daily-limit window no longer affect anything.
func (r *Repository) DeleteOldLoginChallenges(ctx context.Context, tx *gorm.DB, userID uuid.UUID, before time.Time) error {
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND created_at < ?", userID, before).
		Delete(&LoginChallenge{}).Error; err != nil {
		return fmt.Errorf("delete old login challenges: %w", err)
	}
	return nil
}
