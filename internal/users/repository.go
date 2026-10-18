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
		return uuid.Nil, mapUniqueViolation(fmt.Errorf("failed to insert user: %w", err))
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
		return false, fmt.Errorf("error checking whether email is in use: %w", err)
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
		UserID:     v.UserID,
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
		// Re-check the token is still usable under the row lock. Change-email
		// expires old tokens; without this guard an old link could still verify
		// the account after its email was switched to someone else's address.
		Where("hash_code = ? AND verified_at IS NULL AND expires_at > ?", hashedToken, verifiedAt).
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
			"updated_at":        time.Now().UTC(),
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
		return userData, fmt.Errorf("error fetching user by email: %w", result.Error)
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

// GetOAuthIdentity returns the row identified by the provider's immutable
// subject, not by email. Callers should use it before deciding whether a user
// has already authenticated with a provider.
func (r *Repository) GetOAuthIdentity(ctx context.Context, provider, subject string) (OAuthIdentity, error) {
	var identity OAuthIdentity
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_subject = ?", provider, subject).
		First(&identity).Error; err != nil {
		return OAuthIdentity{}, fmt.Errorf("get oauth identity: %w", err)
	}
	return identity, nil
}

// GetOAuthIdentityForUser returns a provider connection owned by one user.
// A user may have at most one identity for each provider.
func (r *Repository) GetOAuthIdentityForUser(ctx context.Context, userID uuid.UUID, provider string) (OAuthIdentity, error) {
	var identity OAuthIdentity
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&identity).Error; err != nil {
		return OAuthIdentity{}, fmt.Errorf("get oauth identity for user: %w", err)
	}
	return identity, nil
}

func (r *Repository) CreateOAuthIdentity(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	provider, subject string,
	providerEmail *string,
	emailVerified bool,
	now time.Time,
) error {
	identity := OAuthIdentity{
		UserID:          userID,
		Provider:        provider,
		ProviderSubject: subject,
		ProviderEmail:   providerEmail,
		EmailVerified:   emailVerified,
		LastUsedAt:      now,
	}
	if err := tx.WithContext(ctx).Create(&identity).Error; err != nil {
		return mapUniqueViolation(fmt.Errorf("create oauth identity: %w", err))
	}
	return nil
}

func (r *Repository) TouchOAuthIdentity(ctx context.Context, provider, subject string, providerEmail *string, emailVerified bool, now time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&OAuthIdentity{}).
		Where("provider = ? AND provider_subject = ?", provider, subject).
		Updates(map[string]any{
			"provider_email": providerEmail,
			"email_verified": emailVerified,
			"last_used_at":   now,
			"updated_at":     now,
		})
	if result.Error != nil {
		return fmt.Errorf("touch oauth identity: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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

func (r *Repository) RevokeRefreshTokens(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error {
	result := tx.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh tokens: %w", result.Error)
	}
	return nil
}

// GetUserProfileByUserID fetches the profile row for a given user.
func (r *Repository) GetUserProfileByUserID(ctx context.Context, userID uuid.UUID) (UserProfile, error) {
	var profile UserProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return UserProfile{}, fmt.Errorf("error fetching profile for user %s: %w", userID, err)
	}
	return profile, nil
}

// ReplaceVerificationToken deletes any unverified token for the user and
// inserts a fresh one in the same transaction.  Use this for resend flows.
func (r *Repository) ReplaceVerificationToken(ctx context.Context, tx *gorm.DB, userID uuid.UUID, hashCode string, expiresAt *time.Time) error {
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND verified_at IS NULL", userID).
		Delete(&UserVerification{}).Error; err != nil {
		return fmt.Errorf("failed to remove old verification tokens: %w", err)
	}

	v := UserVerification{
		UserID:   userID,
		HashCode: hashCode,
	}
	if expiresAt != nil {
		v.ExpiresAt = *expiresAt
	}
	if err := tx.WithContext(ctx).Create(&v).Error; err != nil {
		return fmt.Errorf("failed to insert new verification token: %w", err)
	}
	return nil
}

// One statement against idx_refresh_tokens_user_id, so a login costs a single
// round trip and two concurrent first logins cannot create duplicate rows.
func (r *Repository) UpsertRefreshToken(ctx context.Context, userID uuid.UUID, hashToken string, expiresAt time.Time) (RefreshToken, error) {
	rf := RefreshToken{UserID: userID, TokenHash: hashToken, ExpiresAt: expiresAt}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"token_hash": hashToken,
				"expires_at": expiresAt,
				"is_revoked": false,
				"updated_at": time.Now().UTC(),
			}),
		}).
		Create(&rf)

	if result.Error != nil {
		return RefreshToken{}, fmt.Errorf("upsert refresh token: %w", result.Error)
	}

	return rf, nil
}
