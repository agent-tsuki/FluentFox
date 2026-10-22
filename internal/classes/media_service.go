package classes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/imageutil"
)

var channelAvatarLimits = imageutil.AvatarLimits{
	MinSourcePx: common.AVATAR_MIN_SOURCE_PX,
	MaxSourcePx: common.AVATAR_MAX_SOURCE_PX,
	OutputPx:    common.AVATAR_OUTPUT_PX,
	JPEGQuality: common.AVATAR_JPEG_QUALITY,
}

var channelBannerLimits = imageutil.BannerLimits{
	MinWidthPx:  common.CLASS_BANNER_MIN_WIDTH_PX,
	MaxSourcePx: common.CLASS_BANNER_MAX_SOURCE_PX,
	MaxWidthPx:  common.CLASS_BANNER_MAX_WIDTH_PX,
	JPEGQuality: common.AVATAR_JPEG_QUALITY,
}

// imageCleaner deletes channel images this API uploaded. Custom values can
// also be outside URLs, which are never ours to delete.
type imageCleaner struct {
	store  ObjectStore
	prefix string
	logger *zap.Logger
}

func newImageCleaner(store ObjectStore, uploadBaseURL string, log *zap.Logger) *imageCleaner {
	return &imageCleaner{store: store, prefix: strings.TrimRight(uploadBaseURL, "/") + "/", logger: log}
}

func (c *imageCleaner) URL(key string) string { return c.prefix + key }

// A failed delete only leaves an orphaned file, so it is logged, not returned.
func (c *imageCleaner) Delete(ctx context.Context, imageURL *string) {
	if c.store == nil || imageURL == nil {
		return
	}
	key, ok := strings.CutPrefix(*imageURL, c.prefix)
	if !ok || !strings.HasPrefix(key, common.CLASS_CHANNEL_OBJECT_PREFIX) {
		return
	}
	if err := c.store.Delete(ctx, key); err != nil {
		c.logger.Warn("delete old channel image failed", zap.String("key", key), zap.Error(err))
	}
}

type imageKind struct {
	field     common.ChannelField
	folder    string
	normalize func([]byte) ([]byte, error)
}

var (
	avatarImage = imageKind{
		field: common.ChannelFieldAvatarURL, folder: "avatar",
		normalize: func(b []byte) ([]byte, error) { return imageutil.NormalizeAvatar(b, channelAvatarLimits) },
	}
	bannerImage = imageKind{
		field: common.ChannelFieldBannerURL, folder: "banner",
		normalize: func(b []byte) ([]byte, error) { return imageutil.NormalizeBanner(b, channelBannerLimits) },
	}
)

type ChannelMediaService struct {
	repo   ChannelWriter
	perms  PermissionChecker
	store  ObjectStore
	images *imageCleaner
	logger *zap.Logger
	// Bounds how many images are decoded at once (see CLASS_IMAGE_MAX_CONCURRENT_DECODES).
	decodeSlots chan struct{}
}

// store is nil when R2 is not configured; uploads then fail with 503.
func NewChannelMediaService(repo ChannelWriter, perms PermissionChecker, store ObjectStore, uploadBaseURL string, log *zap.Logger) *ChannelMediaService {
	return &ChannelMediaService{
		repo: repo, perms: perms, store: store, images: newImageCleaner(store, uploadBaseURL, log), logger: log,
		decodeSlots: make(chan struct{}, common.CLASS_IMAGE_MAX_CONCURRENT_DECODES),
	}
}

func (s *ChannelMediaService) UploadAvatar(ctx context.Context, id uuid.UUID, body []byte) (AdminChannel, error) {
	return s.upload(ctx, id, body, avatarImage)
}

func (s *ChannelMediaService) UploadBanner(ctx context.Context, id uuid.UUID, body []byte) (AdminChannel, error) {
	return s.upload(ctx, id, body, bannerImage)
}

func (s *ChannelMediaService) upload(ctx context.Context, id uuid.UUID, body []byte, kind imageKind) (AdminChannel, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CHANNEL_MEDIA, channelRef(id)); err != nil {
		return AdminChannel{}, fmt.Errorf("upload channel %s: %w", kind.folder, err)
	}
	if s.store == nil {
		return AdminChannel{}, exceptions.ErrClassUploadUnavailable
	}
	image, err := s.normalize(ctx, body, kind)
	if err != nil {
		return AdminChannel{}, err
	}

	// A new key per upload, so the year-long CDN cache never shows a stale image.
	key := fmt.Sprintf("%s%s/%s/%s.jpg", common.CLASS_CHANNEL_OBJECT_PREFIX, id, kind.folder, uuid.New())
	if err := s.store.Put(ctx, key, common.AVATAR_CONTENT_TYPE, common.AVATAR_CACHE_CONTROL, image); err != nil {
		return AdminChannel{}, fmt.Errorf("upload channel %s: %w", kind.folder, err)
	}
	newURL := s.images.URL(key)
	before, after, err := updateChannel(ctx, s.repo, id, func(_ *gorm.DB, ch *Channel, cs *changeSet) error {
		p, _ := ch.fieldPair(kind.field)
		cs.Text(customColumn(kind.field), p.Custom, &newURL)
		return nil
	})
	if err != nil {
		s.images.Delete(ctx, &newURL)
		return AdminChannel{}, err
	}
	old, _ := before.fieldPair(kind.field)
	s.images.Delete(ctx, old.Custom)

	s.logger.Info("channel image uploaded", zap.Stringer("channel_id", id), zap.String("kind", kind.folder))
	return toAdminChannel(&after), nil
}

func (s *ChannelMediaService) normalize(ctx context.Context, body []byte, kind imageKind) ([]byte, error) {
	select {
	case s.decodeSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for image decode slot: %w", ctx.Err())
	}
	defer func() { <-s.decodeSlots }()
	image, err := kind.normalize(body)
	switch {
	case errors.Is(err, imageutil.ErrUnsupportedType):
		return nil, exceptions.ErrClassImageTypeInvalid
	case errors.Is(err, imageutil.ErrInvalidImage):
		return nil, exceptions.ErrClassImageInvalid
	case err != nil:
		return nil, fmt.Errorf("normalize channel %s: %w", kind.folder, err)
	}
	return image, nil
}
