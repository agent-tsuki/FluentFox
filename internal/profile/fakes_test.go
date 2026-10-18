package profile

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
)

var errDB = errors.New("connection refused")

// fakeRepo implements every repository interface of this package.
type fakeRepo struct {
	calls []string

	user       users.User
	userErr    error
	profile    users.UserProfile
	profileErr error
	settings   users.UserSettings
	settingErr error
	sessions   []users.RefreshToken

	lockErr        error
	setAvatarErr   error
	avatarPreset   *string
	avatarKey      *string
	profileUpdate  users.ProfileUpdate
	settingsUpdate users.SettingsUpdate

	revokeOK      bool
	revokedID     uuid.UUID
	revokedOthers int64
	locationOK    bool
	savedLocation users.SessionLocation
}

func (f *fakeRepo) Transaction(_ context.Context, fn func(*gorm.DB) error) error {
	if err := fn(nil); err != nil {
		return err
	}
	f.calls = append(f.calls, "Commit")
	return nil
}

func (f *fakeRepo) GetUserByID(context.Context, uuid.UUID) (users.User, error) {
	return f.user, f.userErr
}

func (f *fakeRepo) GetUserProfileByUserID(context.Context, uuid.UUID) (users.UserProfile, error) {
	return f.profile, f.profileErr
}

func (f *fakeRepo) GetUserSettings(context.Context, uuid.UUID) (users.UserSettings, error) {
	return f.settings, f.settingErr
}

func (f *fakeRepo) ListActiveSessions(context.Context, uuid.UUID, time.Time) ([]users.RefreshToken, error) {
	return f.sessions, nil
}

func (f *fakeRepo) LockProfileForUpdate(context.Context, *gorm.DB, uuid.UUID) (users.UserProfile, error) {
	return f.profile, f.lockErr
}

func (f *fakeRepo) UpdateProfile(_ context.Context, _ *gorm.DB, _ uuid.UUID, u users.ProfileUpdate) error {
	f.profileUpdate = u
	return nil
}

func (f *fakeRepo) UpdateUserSettings(_ context.Context, _ *gorm.DB, _ uuid.UUID, u users.SettingsUpdate) error {
	f.settingsUpdate = u
	return nil
}

func (f *fakeRepo) SetAvatar(_ context.Context, _ *gorm.DB, _ uuid.UUID, presetID, objectKey *string) error {
	f.calls = append(f.calls, "SetAvatar")
	f.avatarPreset, f.avatarKey = presetID, objectKey
	return f.setAvatarErr
}

func (f *fakeRepo) RevokeUserSession(_ context.Context, _, sessionID uuid.UUID) (bool, error) {
	f.revokedID = sessionID
	return f.revokeOK, nil
}

func (f *fakeRepo) RevokeOtherSessions(context.Context, uuid.UUID, uuid.UUID) (int64, error) {
	return f.revokedOthers, nil
}

func (f *fakeRepo) UpdateSessionLocation(_ context.Context, _, _ uuid.UUID, loc *users.SessionLocation) (bool, error) {
	f.savedLocation = *loc
	return f.locationOK, nil
}

type fakeStore struct {
	putErr  error
	put     []string
	deleted []string
}

func (f *fakeStore) Put(_ context.Context, key, _, _ string, _ []byte) error {
	f.put = append(f.put, key)
	return f.putErr
}

func (f *fakeStore) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}

func nop() *zap.Logger { return zap.NewNop() }

func ptr[T any](v T) *T { return &v }

func testPNG(t *testing.T, side int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	for y := range side {
		for x := range side {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
