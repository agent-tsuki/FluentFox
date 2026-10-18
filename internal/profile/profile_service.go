package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

type ProfileService struct {
	repo    ProfileRepo
	avatars *AvatarResolver
	logger  *zap.Logger
}

func NewProfileService(repo ProfileRepo, avatars *AvatarResolver, log *zap.Logger) *ProfileService {
	return &ProfileService{repo: repo, avatars: avatars, logger: log}
}

func (s *ProfileService) Get(ctx context.Context, userID, currentSessionID uuid.UUID) (ProfileResponse, error) {
	data, err := s.load(ctx, userID)
	if err != nil {
		return ProfileResponse{}, err
	}
	user, profile, settings, sessions := data.user, data.profile, data.settings, data.sessions

	resp := ProfileResponse{
		UserID:         user.ID,
		Username:       user.Username,
		Email:          user.Email,
		EmailVerified:  user.IsEmailVerified,
		FirstName:      profile.FirstName,
		LastName:       profile.LastName,
		Bio:            profile.Bio,
		NativeLanguage: profile.NativeLanguage,
		CountryCode:    profile.CountryCode,
		City:           profile.City,
		TimeZone:       settings.CurrentTimeZone,
		DailyReminder:  settings.DailyReminder,
		ReminderTime:   displayReminderTime(settings.ReminderTime),
		Avatar:         s.avatars.For(&profile),
		JoinedAt:       user.CreatedAt,
		LastLoginAt:    user.LastLoginAt,
		LastLogin:      latestLogin(sessions, currentSessionID),
	}
	if profile.TargetLevel != nil {
		level := string(*profile.TargetLevel)
		resp.TargetLevel = &level
	}
	return resp, nil
}

type profileData struct {
	user     users.User
	profile  users.UserProfile
	settings users.UserSettings
	sessions []users.RefreshToken
}

// The four reads are independent, so they run at the same time: the request
// then waits for one database round trip instead of four in a row.
func (s *ProfileService) load(ctx context.Context, userID uuid.UUID) (profileData, error) {
	var data profileData
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		user, err := s.repo.GetUserByID(gctx, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrUserNotFound
		}
		data.user = user
		return wrapLoadErr("load user", err)
	})
	g.Go(func() error {
		profile, err := s.repo.GetUserProfileByUserID(gctx, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrProfileNotFound
		}
		data.profile = profile
		return wrapLoadErr("load profile", err)
	})
	g.Go(func() error {
		// Accounts created before settings existed have no row; show the defaults.
		settings, err := s.repo.GetUserSettings(gctx, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		data.settings = settings
		return wrapLoadErr("load settings", err)
	})
	g.Go(func() error {
		sessions, err := s.repo.ListActiveSessions(gctx, userID, time.Now().UTC())
		data.sessions = sessions
		return wrapLoadErr("load sessions", err)
	})
	if err := g.Wait(); err != nil {
		return profileData{}, err //nolint:wrapcheck // each branch already wrapped or returned a sentinel
	}
	return data, nil
}

func wrapLoadErr(step string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("get profile: %s: %w", step, err)
}

type UpdateProfileService struct {
	writer  ProfileWriter
	reader  *ProfileService
	avatars *AvatarResolver
	store   ObjectStore
	logger  *zap.Logger
}

// store may be nil when R2 is not configured; then an old upload is simply
// left in the bucket when a preset replaces it.
func NewUpdateProfileService(writer ProfileWriter, reader *ProfileService, avatars *AvatarResolver, store ObjectStore, log *zap.Logger) *UpdateProfileService {
	return &UpdateProfileService{writer: writer, reader: reader, avatars: avatars, store: store, logger: log}
}

func (s *UpdateProfileService) Update(ctx context.Context, userID, currentSessionID uuid.UUID, req *UpdateProfileRequest) (ProfileResponse, error) {
	if err := s.validate(req); err != nil {
		return ProfileResponse{}, err
	}

	var oldUpload *string
	err := s.writer.Transaction(ctx, func(tx *gorm.DB) error {
		locked, err := s.writer.LockProfileForUpdate(ctx, tx, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrProfileNotFound
		}
		if err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
		if err := s.writer.UpdateProfile(ctx, tx, userID, toProfileUpdate(req)); err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
		if err := s.writer.UpdateUserSettings(ctx, tx, userID, toSettingsUpdate(req)); err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
		if req.AvatarPreset == nil {
			return nil
		}
		// A picked preset replaces an upload, which is deleted after commit.
		oldUpload = locked.ProfileImage
		return s.writer.SetAvatar(ctx, tx, userID, req.AvatarPreset, nil)
	})
	if err != nil {
		return ProfileResponse{}, fmt.Errorf("update profile: %w", err)
	}

	deleteUpload(ctx, s.store, oldUpload, s.logger)
	return s.reader.Get(ctx, userID, currentSessionID)
}

func (s *UpdateProfileService) validate(req *UpdateProfileRequest) error {
	// Required columns: whitespace would pass the length check and then be
	// trimmed to an empty name.
	if isBlank(req.FirstName) || isBlank(req.NativeLanguage) {
		return exceptions.ErrValidation
	}
	if req.TimeZone != nil && *req.TimeZone != "" && !validTimeZone(*req.TimeZone) {
		return exceptions.ErrTimeZoneInvalid
	}
	if req.AvatarPreset != nil && !s.avatars.IsPreset(*req.AvatarPreset) {
		return exceptions.ErrAvatarPresetInvalid
	}
	return nil
}

// time.LoadLocation also accepts "Local", which means the server's zone.
func validTimeZone(name string) bool {
	if name == "UTC" {
		return true
	}
	if !strings.Contains(name, "/") {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

func toProfileUpdate(req *UpdateProfileRequest) users.ProfileUpdate {
	return users.ProfileUpdate{
		FirstName:      trimmed(req.FirstName),
		LastName:       trimmed(req.LastName),
		Bio:            trimmed(req.Bio),
		NativeLanguage: trimmed(req.NativeLanguage),
		CountryCode:    req.CountryCode,
		City:           trimmed(req.City),
		TargetLevel:    req.TargetLevel,
	}
}

func toSettingsUpdate(req *UpdateProfileRequest) users.SettingsUpdate {
	return users.SettingsUpdate{TimeZone: req.TimeZone, DailyReminder: req.DailyReminder, ReminderTime: storedReminderTime(req.ReminderTime)}
}

// Signup stores reminder_time as "15:04:05"; the API speaks "15:04". Writes
// keep the stored format so the column never mixes the two.
const (
	reminderTimeDisplayLen = len("15:04")
	reminderTimeSeconds    = ":00"
)

func storedReminderTime(hhmm *string) *string {
	if hhmm == nil || *hhmm == "" {
		return hhmm
	}
	stored := *hhmm + reminderTimeSeconds
	return &stored
}

func displayReminderTime(stored *string) *string {
	if stored == nil || len(*stored) < reminderTimeDisplayLen {
		return stored
	}
	hhmm := (*stored)[:reminderTimeDisplayLen]
	return &hhmm
}

func isBlank(value *string) bool {
	return value != nil && strings.TrimSpace(*value) == ""
}

func trimmed(value *string) *string {
	if value == nil {
		return nil
	}
	t := strings.TrimSpace(*value)
	return &t
}

// Best effort: a failed delete only leaves an unused file in the bucket.
func deleteUpload(ctx context.Context, store ObjectStore, key *string, log *zap.Logger) {
	if store == nil || key == nil || *key == "" {
		return
	}
	if err := store.Delete(ctx, *key); err != nil {
		log.Warn("delete old avatar upload failed", zap.String("key", *key), zap.Error(err))
	}
}
