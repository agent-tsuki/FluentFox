package users

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VerificationSendStats struct {
	Count  int64
	Oldest *time.Time
	Latest *time.Time
}

func (r *Repository) GetUserForUsername(ctx context.Context, username string) (User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return User{}, fmt.Errorf("error fetching user for username %s: %w", username, err)
	}
	return user, nil
}

func (r *Repository) LockUserForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (User, error) {
	var user User
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		return User{}, fmt.Errorf("error locking user %s: %w", userID, err)
	}
	return user, nil
}

func (r *Repository) HasActiveVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) (bool, error) {
	var count int64
	err := tx.WithContext(ctx).
		Model(&UserVerification{}).
		Where("user_id = ? AND verified_at IS NULL AND expires_at > ?", userID, now).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("error checking active verification for user %s: %w", userID, err)
	}
	return count > 0, nil
}

// The user row is deleted first with the is_email_verified guard in the same
// statement, so a verify that commits concurrently makes this a no-op instead
// of leaving orphaned child rows.
func (r *Repository) DeleteUnverifiedUser(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (bool, error) {
	result := tx.WithContext(ctx).
		Where("id = ? AND is_email_verified = ?", userID, false).
		Delete(&User{})
	if result.Error != nil {
		return false, fmt.Errorf("error deleting unverified user %s: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		return false, nil
	}

	children := []any{&RefreshToken{}, &UserVerification{}, &UserSettings{}, &UserProfile{}}
	for _, model := range children {
		if err := tx.WithContext(ctx).Where("user_id = ?", userID).Delete(model).Error; err != nil {
			return false, fmt.Errorf("error deleting rows of unverified user %s: %w", userID, err)
		}
	}
	return true, nil
}

func (r *Repository) GetVerificationSendStats(ctx context.Context, tx *gorm.DB, userID uuid.UUID, since time.Time) (VerificationSendStats, error) {
	var stats VerificationSendStats
	err := tx.WithContext(ctx).
		Model(&UserVerification{}).
		Select("COUNT(*) AS count, MIN(created_at) AS oldest, MAX(created_at) AS latest").
		Where("user_id = ? AND created_at > ?", userID, since).
		Scan(&stats).Error
	if err != nil {
		return VerificationSendStats{}, fmt.Errorf("error reading verification stats for user %s: %w", userID, err)
	}
	return stats, nil
}

// revokedTokenExpiry is far in the past so no server clock, however skewed,
// can still treat a replaced token as valid.
var revokedTokenExpiry = time.Unix(0, 0).UTC()

const (
	columnExpiresAt = "expires_at"
	columnUpdatedAt = "updated_at"
)

// Rows are kept, not deleted, so the daily send limit can still count them.
func (r *Repository) ExpireActiveVerifications(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) error {
	err := tx.WithContext(ctx).
		Model(&UserVerification{}).
		Where("user_id = ? AND verified_at IS NULL AND "+columnExpiresAt+" > ?", userID, now).
		Updates(map[string]any{columnExpiresAt: revokedTokenExpiry, columnUpdatedAt: now}).Error
	if err != nil {
		return fmt.Errorf("error expiring verifications for user %s: %w", userID, err)
	}
	return nil
}

func (r *Repository) UpdateUnverifiedUserEmail(ctx context.Context, tx *gorm.DB, userID uuid.UUID, email string) (bool, error) {
	result := tx.WithContext(ctx).
		Model(&User{}).
		Where("id = ? AND is_email_verified = ?", userID, false).
		Updates(map[string]any{"email": email, columnUpdatedAt: time.Now().UTC()})
	if result.Error != nil {
		return false, mapUniqueViolation(fmt.Errorf("error updating email for user %s: %w", userID, result.Error))
	}
	return result.RowsAffected > 0, nil
}
