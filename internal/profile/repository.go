package profile

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
)

// ProfileRepo is what the profile domain needs from the users data layer;
// *users.Repository satisfies it.
type ProfileRepo interface {
	GetUserByID(ctx context.Context, userID uuid.UUID) (users.User, error)
	GetUserProfileByUserID(ctx context.Context, userID uuid.UUID) (users.UserProfile, error)
	GetUserSettings(ctx context.Context, userID uuid.UUID) (users.UserSettings, error)
	ListActiveSessions(ctx context.Context, userID uuid.UUID, now time.Time) ([]users.RefreshToken, error)
}

type ProfileWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	LockProfileForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (users.UserProfile, error)
	UpdateProfile(ctx context.Context, tx *gorm.DB, userID uuid.UUID, u users.ProfileUpdate) error
	UpdateUserSettings(ctx context.Context, tx *gorm.DB, userID uuid.UUID, u users.SettingsUpdate) error
	SetAvatar(ctx context.Context, tx *gorm.DB, userID uuid.UUID, presetID, objectKey *string) error
}

type SessionRepo interface {
	ListActiveSessions(ctx context.Context, userID uuid.UUID, now time.Time) ([]users.RefreshToken, error)
	RevokeUserSession(ctx context.Context, userID, sessionID uuid.UUID) (bool, error)
	RevokeOtherSessions(ctx context.Context, userID, keepID uuid.UUID) (int64, error)
	UpdateSessionLocation(ctx context.Context, userID, sessionID uuid.UUID, loc *users.SessionLocation) (bool, error)
}

// ObjectStore holds uploaded avatars; *storage.R2Store satisfies it.
type ObjectStore interface {
	Put(ctx context.Context, key, contentType, cacheControl string, body []byte) error
	Delete(ctx context.Context, key string) error
}
