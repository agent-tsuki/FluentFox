package users

import (
	"context"
	"fmt"
	"time"

	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Transaction runs fn inside a GORM transaction.
// Commit on nil return, rollback on error.
func (r *Repository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *Repository) CreateUser(ctx context.Context, tx *gorm.DB, email, username, passwordHash string, phoneNo *string) (uuid.UUID, error) {
	user := User{
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		PhoneNo:      phoneNo,
	}
	if err := tx.WithContext(ctx).Create(&user).Error; err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert user: %w", err)
	}
	return user.ID, nil
}

func (r *Repository) CreateProfile(ctx context.Context, tx *gorm.DB, userID uuid.UUID, firstName string, lastName *string, nativeLang string) error {
	profile := UserProfile{
		UserID:         userID,
		FirstName:      firstName,
		LastName:       lastName,
		NativeLanguage: nativeLang,
	}
	if err := tx.WithContext(ctx).Create(&profile).Error; err != nil {
		return fmt.Errorf("failed to insert user profile: %w", err)
	}
	return nil
}

func (r *Repository) CreateUsersSettings(ctx context.Context, tx *gorm.DB, userID uuid.UUID, timeZone *string, reminderTime *time.Time) error {
	var rt *string
	if reminderTime != nil {
		s := reminderTime.Format("15:04:05")
		rt = &s
	}
	settings := UserSettings{
		UserID:          userID,
		CurrentTimeZone: timeZone,
		ReminderTime:    rt,
	}
	if err := tx.WithContext(ctx).Create(&settings).Error; err != nil {
		return fmt.Errorf("failed to insert user settings: %w", err)
	}
	return nil
}

func (r *Repository) CreateUserVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID, hashCode string, expiresAt *time.Time) error {
	v := UserVerification{
		UserID:   userID,
		HashCode: hashCode,
	}
	if expiresAt != nil {
		v.ExpiresAt = *expiresAt
	}
	if err := tx.WithContext(ctx).Create(&v).Error; err != nil {
		return fmt.Errorf("failed to insert user verification: %w", err)
	}
	return nil
}

func (r *Repository) GetExistingUserForEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("error fetching user for email %s: %w", email, err)
	}
	return count > 0, nil
}

func (r *Repository) GetExistingUserForUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("error fetching user for username %s: %w", username, err)
	}
	return count > 0, nil
}

func (r *Repository) GetVerificationToken(ctx context.Context, token string) (VerificationModel, error) {
	var v UserVerification
	if err := r.db.WithContext(ctx).Where("hash_code = ?", token).First(&v).Error; err != nil {
		return VerificationModel{}, fmt.Errorf("error fetching token %s: %w", token, err)
	}
	return VerificationModel{
		HashCode:   v.HashCode,
		ExpiresAt:  v.ExpiresAt,
		VerifiedAt: v.VerifiedAt,
	}, nil
}

func (r *Repository) UpdateVerificationToken(ctx context.Context, tx *gorm.DB, verifiedAt time.Time, hashedToken string) (uuid.UUID, error) {
	var updated UserVerification

	result := tx.WithContext(ctx).
		Model(&updated).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "user_id"}}}).
		Where("hash_code = ?", hashedToken).
		Updates(map[string]any{
			"verified_at": verifiedAt,
			"updated_at":  verifiedAt,
		})

	if result.Error != nil {
		return uuid.Nil, fmt.Errorf("error updating token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return uuid.Nil, exceptions.ErrUpdateFail
	}
	return updated.UserID, nil
}

func (r *Repository) UpdateUserForVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error {
	result := tx.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"is_email_verified": true,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("error updating user %s: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		return exceptions.ErrUpdateFail
	}
	return nil
}


func (r *Repository) GetUserForEmail(ctx context.Context, email string) (User, error) {
	var userData User
	result := r.db.WithContext(ctx).
		Model(&userData).Where("email = ?", email).
		First(&userData)
	if result.Error != nil {
		return userData, fmt.Errorf("error fetching user for email %s: %w", email, result.Error)
	}
	return userData, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return User{}, fmt.Errorf("error fetching user by id %s: %w", userID, err)
	}
	return user, nil
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (RefreshToken, error) {
	var rf RefreshToken

	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&rf).Error; err != nil {
		return RefreshToken{}, fmt.Errorf("refresh token not found: %w", err)
	}
	return rf, nil
}
func (r *Repository) RevokeRefreshToken(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh token: %w", result.Error)
	}
	return nil
}

func (r *Repository) UpsertRefreshToken(ctx context.Context, userID uuid.UUID, hashToken string, expiresAt time.Time) (RefreshToken, error) {
	rf := RefreshToken{
		UserID:    userID,
		TokenHash: hashToken,
		ExpiresAt: expiresAt,
	}

	result := r.db.WithContext(ctx).
		Where(RefreshToken{UserID: userID}).
		Assign(RefreshToken{
			TokenHash: hashToken,
			ExpiresAt: expiresAt,
			IsRevoked: false,
		}).
		FirstOrCreate(&rf)

	if result.Error != nil {
		return RefreshToken{}, fmt.Errorf("upsert refresh token: %w", result.Error)
	}

	return rf, nil
}