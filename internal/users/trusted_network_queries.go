package users

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TrustedNetwork struct {
	UserID     uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	IPPrefix   string    `gorm:"column:ip_prefix;primaryKey"`
	VerifiedAt time.Time `gorm:"column:verified_at"`
}

func (TrustedNetwork) TableName() string { return "user_trusted_networks" }

func (r *Repository) HasTrustedNetwork(ctx context.Context, userID uuid.UUID, ipPrefix string, since time.Time) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&TrustedNetwork{}).
		Where("user_id = ? AND ip_prefix = ? AND verified_at >= ?", userID, ipPrefix, since).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("has trusted network: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) UpsertTrustedNetwork(ctx context.Context, tx *gorm.DB, userID uuid.UUID, ipPrefix string, verifiedAt time.Time) error {
	if err := tx.WithContext(ctx).Exec(`
		INSERT INTO user_trusted_networks (user_id, ip_prefix, verified_at)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, ip_prefix) DO UPDATE SET verified_at = EXCLUDED.verified_at`,
		userID, ipPrefix, verifiedAt).Error; err != nil {
		return fmt.Errorf("upsert trusted network: %w", err)
	}
	return nil
}

func (r *Repository) DeleteExpiredTrustedNetworks(ctx context.Context, tx *gorm.DB, userID uuid.UUID, before time.Time) error {
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND verified_at < ?", userID, before).
		Delete(&TrustedNetwork{}).Error; err != nil {
		return fmt.Errorf("delete expired trusted networks: %w", err)
	}
	return nil
}

func (r *Repository) DeleteTrustedNetworks(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error {
	if err := tx.WithContext(ctx).Where("user_id = ?", userID).Delete(&TrustedNetwork{}).Error; err != nil {
		return fmt.Errorf("delete trusted networks: %w", err)
	}
	return nil
}
