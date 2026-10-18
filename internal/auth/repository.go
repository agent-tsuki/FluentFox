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
	GetRefreshTokenByHash(ctx context.Context, hash string) (users.RefreshToken, error)
	RevokeRefreshTokens(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error
	CreateSession(ctx context.Context, tx *gorm.DB, s *users.NewSession, now time.Time) (uuid.UUID, error)
	PruneSessions(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now, revokedBefore time.Time, keepActive int) error
	SetLastLogin(ctx context.Context, tx *gorm.DB, userID uuid.UUID, at time.Time) error
	RotateSession(ctx context.Context, sessionID uuid.UUID, oldHash, newHash string, expiresAt, now time.Time) (bool, error)
	RevokeSessionByID(ctx context.Context, sessionID uuid.UUID) error
	GetLoginChallengeStats(ctx context.Context, tx *gorm.DB, userID uuid.UUID, since time.Time) (users.LoginChallengeStats, error)
	CreateLoginChallenge(ctx context.Context, tx *gorm.DB, c *users.LoginChallenge) error
	LockLoginChallenge(ctx context.Context, tx *gorm.DB, id uuid.UUID) (users.LoginChallenge, error)
	RecordLoginChallengeAttempt(ctx context.Context, tx *gorm.DB, id uuid.UUID, attempts int) error
	ConsumeLoginChallenge(ctx context.Context, tx *gorm.DB, id uuid.UUID, now time.Time) error
	DeleteOldLoginChallenges(ctx context.Context, tx *gorm.DB, userID uuid.UUID, before time.Time) error

	GetUserForUsername(ctx context.Context, username string) (users.User, error)
	LockUserForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (users.User, error)
	HasActiveVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) (bool, error)
	DeleteUnverifiedUser(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (bool, error)
	GetVerificationSendStats(ctx context.Context, tx *gorm.DB, userID uuid.UUID, since time.Time) (users.VerificationSendStats, error)
	ExpireActiveVerifications(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) error
	DeleteUnsentVerification(ctx context.Context, tx *gorm.DB, userID uuid.UUID, hashCode string) error
	UpdateUnverifiedUserEmail(ctx context.Context, tx *gorm.DB, userID uuid.UUID, email string) (bool, error)
	GetPasswordResetForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (users.PasswordReset, error)
	UpsertPasswordResetCode(ctx context.Context, tx *gorm.DB, userID uuid.UUID, codeHash string, codeExpiresAt, sentAt, windowStartedAt time.Time, sentCount int64) error
	RecordPasswordResetCodeAttempt(ctx context.Context, tx *gorm.DB, userID uuid.UUID, attempts int, now time.Time) error
	SetPasswordResetLink(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tokenHash string, expiresAt, now time.Time) error
	FindPasswordResetUserByToken(ctx context.Context, tokenHash string) (uuid.UUID, error)
	ConsumePasswordReset(ctx context.Context, tx *gorm.DB, userID uuid.UUID, tokenHash, passwordHash string, now time.Time) (bool, error)
}
