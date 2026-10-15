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

	children := []any{&RefreshToken{}, &UserVerification{}, &PasswordReset{}, &UserSettings{}, &UserProfile{}}
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

func (r *Repository) GetPasswordResetForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (PasswordReset, error) {
	var reset PasswordReset
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("user_id = ?", userID).
		First(&reset).Error
	if err != nil {
		return PasswordReset{}, fmt.Errorf("get password reset for user %s: %w", userID, err)
	}
	return reset, nil
}

func (r *Repository) UpsertPasswordResetCode(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	codeHash string,
	codeExpiresAt, sentAt, windowStartedAt time.Time,
	sentCount int64,
) error {
	reset := PasswordReset{
		UserID:            userID,
		CodeHash:          codeHash,
		CodeExpiresAt:     codeExpiresAt,
		CodeLastSentAt:    sentAt,
		CodeWindowStarted: windowStartedAt,
		CodeSentCount:     sentCount,
	}
	updates := map[string]any{
		"code_hash":              codeHash,
		"code_expires_at":        codeExpiresAt,
		"code_verified_at":       nil,
		"code_attempts":          0,
		"code_last_sent_at":      sentAt,
		"code_window_started_at": windowStartedAt,
		"code_sent_count":        sentCount,
		"reset_token_hash":       nil,
		"reset_token_expires_at": nil,
		"reset_token_used_at":    nil,
		"updated_at":             sentAt,
	}
	if err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(updates),
		}).
		Create(&reset).Error; err != nil {
		return fmt.Errorf("upsert password reset code: %w", err)
	}
	return nil
}

func (r *Repository) RecordPasswordResetCodeAttempt(ctx context.Context, tx *gorm.DB, userID uuid.UUID, attempts int, now time.Time) error {
	if err := tx.WithContext(ctx).
		Model(&PasswordReset{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{"code_attempts": attempts, "updated_at": now}).Error; err != nil {
		return fmt.Errorf("record password reset code attempt: %w", err)
	}
	return nil
}

func (r *Repository) SetPasswordResetLink(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tokenHash string, expiresAt, now time.Time) error {
	if err := tx.WithContext(ctx).
		Model(&PasswordReset{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"code_verified_at":       now,
			"reset_token_hash":       tokenHash,
			"reset_token_expires_at": expiresAt,
			"reset_token_used_at":    nil,
			"updated_at":             now,
		}).Error; err != nil {
		return fmt.Errorf("set password reset link: %w", err)
	}
	return nil
}

func (r *Repository) FindPasswordResetUserByToken(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	var reset PasswordReset
	if err := r.db.WithContext(ctx).Where("reset_token_hash = ?", tokenHash).First(&reset).Error; err != nil {
		return uuid.Nil, fmt.Errorf("find password reset token: %w", err)
	}
	return reset.UserID, nil
}

func (r *Repository) ConsumePasswordReset(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	tokenHash, passwordHash string,
	now time.Time,
) (bool, error) {
	result := tx.WithContext(ctx).
		Model(&PasswordReset{}).
		Where("user_id = ? AND reset_token_hash = ? AND reset_token_used_at IS NULL AND reset_token_expires_at > ?", userID, tokenHash, now).
		Updates(map[string]any{"reset_token_used_at": now, "updated_at": now})
	if result.Error != nil {
		return false, fmt.Errorf("consume password reset token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return false, nil
	}

	if err := tx.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"password_hash": passwordHash, "updated_at": now}).Error; err != nil {
		return false, fmt.Errorf("update password after reset: %w", err)
	}
	if err := r.RevokeRefreshTokens(ctx, tx, userID); err != nil {
		return false, err
	}
	return true, nil
}
