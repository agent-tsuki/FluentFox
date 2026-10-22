package classes

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/youtube"
)

// Turns a YouTube failure into what the editor sees; the cause stays for the log.
func mapYouTubeErr(err error) error {
	if errors.Is(err, youtube.ErrNotFound) {
		return exceptions.ErrYouTubeSourceNotFound
	}
	return exceptions.WithCause(exceptions.ErrYouTubeSyncFailed, err)
}

type ChannelSyncService struct {
	repo   ChannelWriter
	yt     YouTubeChannels
	perms  PermissionChecker
	logger *zap.Logger
}

// yt is nil when YOUTUBE_API_KEY is not set; sync then answers 503.
func NewChannelSyncService(repo ChannelWriter, yt YouTubeChannels, perms PermissionChecker, log *zap.Logger) *ChannelSyncService {
	return &ChannelSyncService{repo: repo, yt: yt, perms: perms, logger: log}
}

// Sync writes only the *_youtube columns and the counts, so an editor's
// custom values survive every sync.
func (s *ChannelSyncService) Sync(ctx context.Context, id uuid.UUID) (AdminChannel, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CLASSES_SYNC, channelRef(id)); err != nil {
		return AdminChannel{}, fmt.Errorf("sync channel: %w", err)
	}
	if s.yt == nil {
		return AdminChannel{}, exceptions.ErrYouTubeSyncUnavailable
	}
	current, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		return AdminChannel{}, notFoundAs(err, exceptions.ErrChannelNotFound, "sync channel")
	}
	lookup := youtube.ChannelLookup{ID: deref(current.YouTubeChannelID), Handle: deref(current.Handle)}
	if lookup.ID == "" && lookup.Handle == "" {
		return AdminChannel{}, exceptions.ErrYouTubeSourceMissing
	}

	// YouTube is called before the transaction so no row lock is held while waiting on it.
	callCtx, cancel := context.WithTimeout(ctx, common.YOUTUBE_SYNC_BUDGET)
	defer cancel()
	remote, err := s.yt.Channel(callCtx, lookup)
	if err != nil {
		return AdminChannel{}, mapYouTubeErr(err)
	}

	now := time.Now()
	_, after, err := updateChannel(ctx, s.repo, id, func(_ *gorm.DB, ch *Channel, cs *changeSet) error {
		applyYouTubeChannel(ch, &remote, cs)
		cs.columns["last_synced_at"] = now
		return nil
	})
	if err != nil {
		return AdminChannel{}, err
	}
	s.logger.Info("class channel synced", zap.Stringer("channel_id", id))
	return toAdminChannel(&after), nil
}

func applyYouTubeChannel(ch *Channel, remote *youtube.Channel, cs *changeSet) {
	cs.Text(youtubeColumn(common.ChannelFieldName), ch.NameYouTube, nonEmpty(remote.Title))
	cs.Text(youtubeColumn(common.ChannelFieldDescription), ch.DescriptionYouTube, nonEmpty(remote.Description))
	cs.Text(youtubeColumn(common.ChannelFieldAvatarURL), ch.AvatarURLYouTube, nonEmpty(remote.AvatarURL))
	cs.Text(youtubeColumn(common.ChannelFieldBannerURL), ch.BannerURLYouTube, nonEmpty(remote.BannerURL))
	cs.Text(youtubeColumn(common.ChannelFieldIntroVideoID), ch.IntroVideoIDYouTube, nonEmpty(remote.TrailerVideoID))
	cs.Value("subscriber_count", ch.SubscriberCount, remote.SubscriberCount)
	cs.Value("video_count", ch.VideoCount, remote.VideoCount)
	// A channel found by handle learns its stable id, which later syncs prefer.
	if ch.YouTubeChannelID == nil && remote.ID != "" {
		cs.Text("youtube_channel_id", nil, &remote.ID)
	}
}

type PlaylistSyncService struct {
	repo   PlaylistWriter
	yt     YouTubePlaylists
	reader *CourseAdminGetService
	perms  PermissionChecker
	logger *zap.Logger
}

// yt is nil when YOUTUBE_API_KEY is not set; sync then answers 503.
func NewPlaylistSyncService(
	repo PlaylistWriter, yt YouTubePlaylists, reader *CourseAdminGetService, perms PermissionChecker, log *zap.Logger,
) *PlaylistSyncService {
	return &PlaylistSyncService{repo: repo, yt: yt, reader: reader, perms: perms, logger: log}
}

// Sync replaces the course's videos with the playlist as YouTube has it now.
// Private and deleted videos have no duration and are left out.
func (s *PlaylistSyncService) Sync(ctx context.Context, courseID uuid.UUID) (AdminCourse, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CLASSES_SYNC, courseRef(courseID)); err != nil {
		return AdminCourse{}, fmt.Errorf("sync playlist: %w", err)
	}
	if s.yt == nil {
		return AdminCourse{}, exceptions.ErrYouTubeSyncUnavailable
	}
	course, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return AdminCourse{}, notFoundAs(err, exceptions.ErrClassNotFound, "sync playlist")
	}
	if course.PlaylistID == nil || *course.PlaylistID == "" {
		return AdminCourse{}, exceptions.ErrYouTubeSourceMissing
	}
	videos, err := s.fetch(ctx, courseID, *course.PlaylistID)
	if err != nil {
		return AdminCourse{}, err
	}
	if err := s.save(ctx, courseID, videos); err != nil {
		return AdminCourse{}, err
	}
	s.logger.Info("class playlist synced", zap.Stringer("course_id", courseID), zap.Int("videos", len(videos)))
	return s.reader.load(ctx, courseID)
}

func (s *PlaylistSyncService) save(ctx context.Context, courseID uuid.UUID, videos []Video) error {
	total := 0
	for _, v := range videos {
		total += v.DurationSec
	}
	now := time.Now()
	return updateCourse(ctx, s.repo, courseID, func(tx *gorm.DB, c *Course, cs *changeSet) error {
		if err := s.repo.ReplaceVideos(ctx, tx, courseID, videos); err != nil {
			return fmt.Errorf("sync playlist: %w", err)
		}
		cs.Note(logFieldVideosSynced, strconv.Itoa(len(videos)))
		cs.Value("total_duration_sec", c.TotalDurationSec, total)
		// The thumbnail is an editor's field; sync only fills it while it is empty.
		if c.ThumbnailURL == nil && len(videos) > 0 && videos[0].ThumbnailURL != "" {
			cs.Text("thumbnail_url", nil, &videos[0].ThumbnailURL)
		}
		cs.columns["last_synced_at"] = now
		return nil
	})
}

func (s *PlaylistSyncService) fetch(ctx context.Context, courseID uuid.UUID, playlistID string) ([]Video, error) {
	callCtx, cancel := context.WithTimeout(ctx, common.YOUTUBE_SYNC_BUDGET)
	defer cancel()
	items, err := s.yt.PlaylistItems(callCtx, playlistID, common.YOUTUBE_PLAYLIST_MAX_PAGES)
	if err != nil {
		return nil, mapYouTubeErr(err)
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.VideoID)
	}
	durations, err := s.yt.VideoDurations(callCtx, ids)
	if err != nil {
		return nil, mapYouTubeErr(err)
	}
	videos := make([]Video, 0, len(items))
	for _, it := range items {
		d, ok := durations[it.VideoID]
		if !ok {
			continue
		}
		videos = append(videos, Video{
			CourseID: courseID, YouTubeVideoID: it.VideoID, Title: it.Title,
			ThumbnailURL: it.ThumbnailURL, DurationSec: int(d / time.Second), Position: len(videos),
		})
	}
	return videos, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// YouTube sends "" for a missing value; nil keeps the column empty.
func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
