package auth

import (
	"context"
	"time"

	"github.com/fluentfox/api/internal/users"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepo is the interface the auth domain requires from the users data layer.
// *users.Repository satisfies this interface implicitly — no changes needed there.
type UserRepo interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	CreateUser(ctx context.Context, tx *gorm.DB, email, username, passwordHash string, phoneNo *string) (uuid.UUID, error)
	CreateProfile(ctx context.Context, tx *gorm.DB, userID uuid.UUID, firstName string, lastName *string, nativeLang string) error
	CreateUsersSettings(ctx context.Context, tx *gorm.DB, userID uuid.UUID, timeZone *string, reminderTime *time.Time) error
	CreateUserVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID, hashCode string, expiresAt *time.Time) error
	GetExistingUserForEmail(ctx context.Context, email string) (bool, error)
	GetExistingUserForUsername(ctx context.Context, username string) (bool, error)
	GetVerificationToken(ctx context.Context, token string) (users.VerificationModel, error)
	UpdateVerificationToken(ctx context.Context, tx *gorm.DB, verifiedAt time.Time, token string) (uuid.UUID, error)
	UpdateUserForVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error
	GetUserForEmail(ctx context.Context, email string) (users.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (users.User, error)
	GetUserProfileByUserID(ctx context.Context, userID uuid.UUID) (users.UserProfile, error)
	ReplaceVerificationToken(ctx context.Context, tx *gorm.DB, userID uuid.UUID, hashCode string, expiresAt *time.Time) error
	UpsertRefreshToken(ctx context.Context, userID uuid.UUID, hashToken string, expiresAt time.Time) (users.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, hash string) (users.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, userID uuid.UUID) error

	GetUserForUsername(ctx context.Context, username string) (users.User, error)
	LockUserForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (users.User, error)
	HasActiveVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) (bool, error)
	DeleteUnverifiedUser(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (bool, error)
	GetVerificationSendStats(ctx context.Context, tx *gorm.DB, userID uuid.UUID, since time.Time) (users.VerificationSendStats, error)
	ExpireActiveVerifications(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) error
	UpdateUnverifiedUserEmail(ctx context.Context, tx *gorm.DB, userID uuid.UUID, email string) (bool, error)
}
