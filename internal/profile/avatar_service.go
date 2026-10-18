package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/imageutil"
)

var avatarLimits = imageutil.AvatarLimits{
	MinSourcePx: common.AVATAR_MIN_SOURCE_PX,
	MaxSourcePx: common.AVATAR_MAX_SOURCE_PX,
	OutputPx:    common.AVATAR_OUTPUT_PX,
	JPEGQuality: common.AVATAR_JPEG_QUALITY,
}

type AvatarService struct {
	writer  ProfileWriter
	store   ObjectStore
	avatars *AvatarResolver
	logger  *zap.Logger
	// Bounds how many images are decoded at once (see AVATAR_MAX_CONCURRENT_DECODES).
	decodeSlots chan struct{}
}

// store is nil when R2 is not configured; uploads then fail with 503 while
// presets keep working.
func NewAvatarService(writer ProfileWriter, store ObjectStore, avatars *AvatarResolver, log *zap.Logger) *AvatarService {
	return &AvatarService{
		writer: writer, store: store, avatars: avatars, logger: log,
		decodeSlots: make(chan struct{}, common.AVATAR_MAX_CONCURRENT_DECODES),
	}
}

func (s *AvatarService) Upload(ctx context.Context, userID uuid.UUID, body []byte) (AvatarResponse, error) {
	if s.store == nil {
		return AvatarResponse{}, exceptions.ErrAvatarUploadUnavailable
	}
	image, err := s.normalize(ctx, body)
	switch {
	case errors.Is(err, imageutil.ErrUnsupportedType):
		return AvatarResponse{}, exceptions.ErrAvatarTypeInvalid
	case errors.Is(err, imageutil.ErrInvalidImage):
		return AvatarResponse{}, exceptions.ErrAvatarImageInvalid
	case err != nil:
		return AvatarResponse{}, fmt.Errorf("upload avatar: %w", err)
	}

	// A new key per upload, so the year-long CDN cache never shows a stale image.
	key := fmt.Sprintf("%s%s/%s.jpg", common.AVATAR_OBJECT_PREFIX, userID, uuid.New())
	if err := s.store.Put(ctx, key, common.AVATAR_CONTENT_TYPE, common.AVATAR_CACHE_CONTROL, image); err != nil {
		return AvatarResponse{}, fmt.Errorf("upload avatar: %w", err)
	}

	oldUpload, err := s.replaceWith(ctx, userID, nil, &key, nil)
	if err != nil {
		deleteUpload(ctx, s.store, &key, s.logger)
		return AvatarResponse{}, err
	}
	deleteUpload(ctx, s.store, oldUpload, s.logger)

	s.logger.Info("avatar uploaded", zap.Stringer("user_id", userID))
	return s.avatars.For(&users.UserProfile{ProfileImage: &key}), nil
}

func (s *AvatarService) normalize(ctx context.Context, body []byte) ([]byte, error) {
	select {
	case s.decodeSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for avatar decode slot: %w", ctx.Err())
	}
	defer func() { <-s.decodeSlots }()
	return imageutil.NormalizeAvatar(body, avatarLimits) //nolint:wrapcheck // sentinel errors are matched by the caller
}

// Remove drops the user's own choice, so the Google/X picture or the default
// shows again.
func (s *AvatarService) Remove(ctx context.Context, userID uuid.UUID) (AvatarResponse, error) {
	var current users.UserProfile
	oldUpload, err := s.replaceWith(ctx, userID, nil, nil, &current)
	if err != nil {
		return AvatarResponse{}, err
	}
	deleteUpload(ctx, s.store, oldUpload, s.logger)

	current.ProfileImage, current.AvatarPreset = nil, nil
	return s.avatars.For(&current), nil
}

// replaceWith sets the avatar under the profile lock and returns the upload it
// replaced; current, if non-nil, receives the profile as it was. The lock
// stops two uploads from each deleting the other's file.
func (s *AvatarService) replaceWith(ctx context.Context, userID uuid.UUID, presetID, objectKey *string, current *users.UserProfile) (*string, error) {
	var oldUpload *string
	err := s.writer.Transaction(ctx, func(tx *gorm.DB) error {
		locked, err := s.writer.LockProfileForUpdate(ctx, tx, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrProfileNotFound
		}
		if err != nil {
			return fmt.Errorf("set avatar: %w", err)
		}
		if current != nil {
			*current = locked
		}
		oldUpload = locked.ProfileImage
		if err := s.writer.SetAvatar(ctx, tx, userID, presetID, objectKey); err != nil {
			return fmt.Errorf("set avatar: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("set avatar: %w", err)
	}
	return oldUpload, nil
}
