package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProfileUpdate changes only the non-nil fields; an empty string clears an
// optional one. users_profile.user_id is text (built by the old AutoMigrate).
type ProfileUpdate struct {
	FirstName      *string
	LastName       *string
	Bio            *string
	NativeLanguage *string
	CountryCode    *string
	City           *string
	TargetLevel    *string
}

// SettingsUpdate changes only the non-nil fields of users_settings.
type SettingsUpdate struct {
	TimeZone      *string
	DailyReminder *bool
	ReminderTime  *string
}

func (r *Repository) GetUserSettings(ctx context.Context, userID uuid.UUID) (UserSettings, error) {
	var settings UserSettings
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID.String()).First(&settings).Error; err != nil {
		return UserSettings{}, fmt.Errorf("get user settings: %w", err)
	}
	return settings, nil
}

func (r *Repository) LockProfileForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (UserProfile, error) {
	var profile UserProfile
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("user_id = ?", userID.String()).
		First(&profile).Error; err != nil {
		return UserProfile{}, fmt.Errorf("lock profile: %w", err)
	}
	return profile, nil
}

func (r *Repository) UpdateProfile(ctx context.Context, tx *gorm.DB, userID uuid.UUID, u ProfileUpdate) error {
	changes := map[string]any{}
	setRequired(changes, "first_name", u.FirstName)
	setRequired(changes, "native_language", u.NativeLanguage)
	setOptional(changes, "last_name", u.LastName)
	setOptional(changes, "bio", u.Bio)
	setOptional(changes, "country_code", u.CountryCode)
	setOptional(changes, "city", u.City)
	setOptional(changes, "target_level", u.TargetLevel)
	if len(changes) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Model(&UserProfile{}).
		Where("user_id = ?", userID.String()).Updates(changes).Error; err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

// Older accounts may have no settings row, so a missing row is created.
func (r *Repository) UpdateUserSettings(ctx context.Context, tx *gorm.DB, userID uuid.UUID, u SettingsUpdate) error {
	changes := map[string]any{}
	setOptional(changes, "current_time_zone", u.TimeZone)
	setOptional(changes, "reminder_time", u.ReminderTime)
	if u.DailyReminder != nil {
		changes["daily_reminder"] = *u.DailyReminder
	}
	if len(changes) == 0 {
		return nil
	}

	result := tx.WithContext(ctx).Model(&UserSettings{}).Where("user_id = ?", userID.String()).Updates(changes)
	if result.Error != nil {
		return fmt.Errorf("update user settings: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		return nil
	}
	settings := UserSettings{UserID: userID, CurrentTimeZone: nullIfEmptyPtr(u.TimeZone), ReminderTime: nullIfEmptyPtr(u.ReminderTime)}
	if u.DailyReminder != nil {
		settings.DailyReminder = *u.DailyReminder
	}
	if err := tx.WithContext(ctx).Create(&settings).Error; err != nil {
		return fmt.Errorf("create user settings: %w", err)
	}
	return nil
}

// SetAvatar records the user's own choice: an uploaded object key or a
// preset id. Setting one clears the other.
func (r *Repository) SetAvatar(ctx context.Context, tx *gorm.DB, userID uuid.UUID, presetID, objectKey *string) error {
	if err := tx.WithContext(ctx).Model(&UserProfile{}).Where("user_id = ?", userID.String()).
		Updates(map[string]any{"avatar_preset": presetID, "profile_image": objectKey}).Error; err != nil {
		return fmt.Errorf("set avatar: %w", err)
	}
	return nil
}

// The latest Google/X picture is always stored; whether it is shown is
// decided at read time, after any upload or preset the user picked.
func (r *Repository) SetOAuthImage(ctx context.Context, userID uuid.UUID, imageURL string) error {
	if err := r.db.WithContext(ctx).Model(&UserProfile{}).Where("user_id = ?", userID.String()).
		Update("oauth_image_url", imageURL).Error; err != nil {
		return fmt.Errorf("set oauth image: %w", err)
	}
	return nil
}

func setRequired(changes map[string]any, column string, value *string) {
	if value != nil {
		changes[column] = *value
	}
}

func setOptional(changes map[string]any, column string, value *string) {
	if value != nil {
		changes[column] = nullIfEmpty(*value)
	}
}

func nullIfEmptyPtr(value *string) *string {
	if value == nil {
		return nil
	}
	return nullIfEmpty(*value)
}
