package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
)

// fakeUserRepo embeds the UserRepo interface (nil) so a test only implements
// what it uses; calling anything else panics, which flags an unexpected call.
type fakeUserRepo struct {
	UserRepo

	calls []string

	byEmail    map[string]users.User
	emailErr   error
	byUsername map[string]users.User

	byID    users.User
	byIDErr error

	profile    users.UserProfile
	profileErr error

	locked  users.User
	lockErr error

	hasActive    bool
	hasActiveErr error

	deleted    bool
	deleteErr  error
	deletedIDs []uuid.UUID

	stats    users.VerificationSendStats
	statsErr error

	expireErr    error
	createVerErr error

	updateEmailOK  bool
	updateEmailErr error
	updatedEmail   string

	createdEmail    string
	createdUsername string
	// createUserErrs is consumed one per CreateUser call, to simulate a clash then success.
	createUserErrs   []error
	createdUsernames []string

	lookedUpEmail string

	refresh       users.RefreshToken
	refreshErr    error
	upsertErr     error
	upsertedHash  string
	upsertedCalls int
	revokeErr     error
	revokedIDs    []uuid.UUID

	passwordReset        users.PasswordReset
	passwordResetErr     error
	storedCodeHash       string
	storedCodeExpiresAt  time.Time
	storedSentCount      int64
	storedCodeAttempts   int
	storedLinkHash       string
	storedLinkExpiresAt  time.Time
	findResetUserID      uuid.UUID
	findResetErr         error
	consumeErr           error
	consumed             bool
	consumedPasswordHash string
}

func (f *fakeUserRepo) record(name string) { f.calls = append(f.calls, name) }

func (f *fakeUserRepo) Transaction(_ context.Context, fn func(*gorm.DB) error) error {
	f.record("Transaction")
	if err := fn(nil); err != nil {
		return err
	}
	f.record("Commit")
	return nil
}

func (f *fakeUserRepo) GetUserForEmail(_ context.Context, email string) (users.User, error) {
	f.lookedUpEmail = email
	if f.emailErr != nil {
		return users.User{}, f.emailErr
	}
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return users.User{}, gorm.ErrRecordNotFound
}

func (f *fakeUserRepo) GetUserForUsername(_ context.Context, name string) (users.User, error) {
	if u, ok := f.byUsername[name]; ok {
		return u, nil
	}
	return users.User{}, gorm.ErrRecordNotFound
}

func (f *fakeUserRepo) GetExistingUserForUsername(context.Context, string) (bool, error) {
	return false, nil
}

func (f *fakeUserRepo) GetUserByID(context.Context, uuid.UUID) (users.User, error) {
	return f.byID, f.byIDErr
}

func (f *fakeUserRepo) GetUserProfileByUserID(context.Context, uuid.UUID) (users.UserProfile, error) {
	return f.profile, f.profileErr
}

func (f *fakeUserRepo) LockUserForUpdate(context.Context, *gorm.DB, uuid.UUID) (users.User, error) {
	f.record("LockUserForUpdate")
	return f.locked, f.lockErr
}

func (f *fakeUserRepo) HasActiveVerification(context.Context, *gorm.DB, uuid.UUID, time.Time) (bool, error) {
	return f.hasActive, f.hasActiveErr
}

func (f *fakeUserRepo) DeleteUnverifiedUser(_ context.Context, _ *gorm.DB, id uuid.UUID) (bool, error) {
	f.record("DeleteUnverifiedUser")
	f.deletedIDs = append(f.deletedIDs, id)
	return f.deleted, f.deleteErr
}

func (f *fakeUserRepo) GetVerificationSendStats(context.Context, *gorm.DB, uuid.UUID, time.Time) (users.VerificationSendStats, error) {
	return f.stats, f.statsErr
}

func (f *fakeUserRepo) DeleteUnsentVerification(context.Context, *gorm.DB, uuid.UUID, string) error {
	f.record("DeleteUnsentVerification")
	return nil
}

func (f *fakeUserRepo) ExpireActiveVerifications(context.Context, *gorm.DB, uuid.UUID, time.Time) error {
	f.record("ExpireActiveVerifications")
	return f.expireErr
}

func (f *fakeUserRepo) CreateUserVerification(context.Context, *gorm.DB, uuid.UUID, string, *time.Time) error {
	f.record("CreateUserVerification")
	return f.createVerErr
}

func (f *fakeUserRepo) UpdateUnverifiedUserEmail(_ context.Context, _ *gorm.DB, _ uuid.UUID, email string) (bool, error) {
	f.record("UpdateUnverifiedUserEmail")
	f.updatedEmail = email
	return f.updateEmailOK, f.updateEmailErr
}

func (f *fakeUserRepo) CreateUser(_ context.Context, _ *gorm.DB, email, username, _ string, _ *string) (uuid.UUID, error) {
	f.record("CreateUser")
	f.createdEmail, f.createdUsername = email, username
	f.createdUsernames = append(f.createdUsernames, username)
	if len(f.createUserErrs) > 0 {
		err := f.createUserErrs[0]
		f.createUserErrs = f.createUserErrs[1:]
		if err != nil {
			return uuid.Nil, err
		}
	}
	return uuid.New(), nil
}

func (f *fakeUserRepo) CreateProfile(context.Context, *gorm.DB, uuid.UUID, string, *string, string) error {
	f.record("CreateProfile")
	return nil
}

func (f *fakeUserRepo) CreateUsersSettings(context.Context, *gorm.DB, uuid.UUID, *string, *time.Time) error {
	f.record("CreateUsersSettings")
	return nil
}

func (f *fakeUserRepo) UpsertRefreshToken(_ context.Context, userID uuid.UUID, hash string, exp time.Time) (users.RefreshToken, error) {
	f.upsertedCalls++
	f.upsertedHash = hash
	return users.RefreshToken{UserID: userID, TokenHash: hash, ExpiresAt: exp}, f.upsertErr
}

func (f *fakeUserRepo) GetRefreshTokenByHash(context.Context, string) (users.RefreshToken, error) {
	return f.refresh, f.refreshErr
}

func (f *fakeUserRepo) RevokeRefreshToken(_ context.Context, userID uuid.UUID) error {
	f.revokedIDs = append(f.revokedIDs, userID)
	return f.revokeErr
}

func (f *fakeUserRepo) RevokeRefreshTokens(context.Context, *gorm.DB, uuid.UUID) error {
	return f.revokeErr
}

func (f *fakeUserRepo) GetPasswordResetForUpdate(context.Context, *gorm.DB, uuid.UUID) (users.PasswordReset, error) {
	return f.passwordReset, f.passwordResetErr
}

func (f *fakeUserRepo) UpsertPasswordResetCode(_ context.Context, _ *gorm.DB, _ uuid.UUID, codeHash string, expiresAt, _ time.Time, _ time.Time, sentCount int64) error {
	f.record("UpsertPasswordResetCode")
	f.storedCodeHash = codeHash
	f.storedCodeExpiresAt = expiresAt
	f.storedSentCount = sentCount
	return nil
}

func (f *fakeUserRepo) RecordPasswordResetCodeAttempt(_ context.Context, _ *gorm.DB, _ uuid.UUID, attempts int, _ time.Time) error {
	f.record("RecordPasswordResetCodeAttempt")
	f.storedCodeAttempts = attempts
	return nil
}

func (f *fakeUserRepo) SetPasswordResetLink(_ context.Context, _ *gorm.DB, _ uuid.UUID, tokenHash string, expiresAt, _ time.Time) error {
	f.record("SetPasswordResetLink")
	f.storedLinkHash = tokenHash
	f.storedLinkExpiresAt = expiresAt
	return nil
}

func (f *fakeUserRepo) FindPasswordResetUserByToken(context.Context, string) (uuid.UUID, error) {
	return f.findResetUserID, f.findResetErr
}

func (f *fakeUserRepo) ConsumePasswordReset(_ context.Context, _ *gorm.DB, _ uuid.UUID, _ string, passwordHash string, _ time.Time) (bool, error) {
	f.record("ConsumePasswordReset")
	f.consumedPasswordHash = passwordHash
	return f.consumed, f.consumeErr
}

type fakeEmailChecker struct {
	err     error
	checked []string
}

func (f *fakeEmailChecker) Check(_ context.Context, email string) error {
	f.checked = append(f.checked, email)
	return f.err
}

type sentMail struct{ to, name, token string }

type fakeMailer struct {
	err  error
	sent []sentMail
	// onSend lets a test record call ordering against the repo.
	onSend func()

	resetCodeErr error
	resetLinkErr error
	resetCodes   []sentMail
	resetLinks   []sentMail
}

func (f *fakeMailer) SendVerificationEmail(_ context.Context, to, name, rawToken string) error {
	if f.onSend != nil {
		f.onSend()
	}
	f.sent = append(f.sent, sentMail{to, name, rawToken})
	return f.err
}

func (f *fakeMailer) SendPasswordResetCode(_ context.Context, to, name, code string) error {
	f.resetCodes = append(f.resetCodes, sentMail{to: to, name: name, token: code})
	return f.resetCodeErr
}

func (f *fakeMailer) SendPasswordResetLink(_ context.Context, to, name, rawToken string) error {
	f.resetLinks = append(f.resetLinks, sentMail{to: to, name: name, token: rawToken})
	return f.resetLinkErr
}

func nopLogger() *zap.Logger { return zap.NewNop() }
