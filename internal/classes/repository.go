package classes

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/youtube"
)

// PermissionChecker is satisfied by *access.Checker; Require returns
// exceptions.ErrForbidden when the signed-in user lacks perm on res.
type PermissionChecker interface {
	Require(ctx context.Context, perm common.Permission, res common.ResourceRef) error
}

// GrantLister is satisfied by *access.Checker. all is true for super-admins
// and global classes grants; ids are channels the user holds a classes grant on.
type GrantLister interface {
	EditableChannelIDs(ctx context.Context) (all bool, ids []string, err error)
}

// CatalogReader and the interfaces below are satisfied by *Repository.
type CatalogReader interface {
	ListVisibleCourses(ctx context.Context) ([]CourseListing, error)
	GetCourseBySlug(ctx context.Context, slug string) (Course, error)
	GetChannel(ctx context.Context, id uuid.UUID) (Channel, error)
	ListVideos(ctx context.Context, courseID uuid.UUID) ([]Video, error)
}

type ChannelLister interface {
	// ids nil lists every channel.
	ListChannels(ctx context.Context, ids []uuid.UUID) ([]Channel, error)
}

type ChannelReader interface {
	GetChannel(ctx context.Context, id uuid.UUID) (Channel, error)
}

type ChannelWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	GetChannel(ctx context.Context, id uuid.UUID) (Channel, error)
	LockChannel(ctx context.Context, tx *gorm.DB, id uuid.UUID) (Channel, error)
	CreateChannel(ctx context.Context, tx *gorm.DB, ch *Channel) error
	UpdateChannel(ctx context.Context, tx *gorm.DB, id uuid.UUID, columns map[string]any) error
	AppendChangeLog(ctx context.Context, tx *gorm.DB, rows []ChangeLog) error
}

type HistoryReader interface {
	ListChannelHistory(ctx context.Context, channelID uuid.UUID, limit int) ([]ChangeLogEntry, error)
}

type CourseAdminReader interface {
	// channelID nil lists every course, hidden ones included.
	ListCourses(ctx context.Context, channelID *uuid.UUID) ([]CourseListing, error)
	GetCourse(ctx context.Context, id uuid.UUID) (Course, error)
	GetChannel(ctx context.Context, id uuid.UUID) (Channel, error)
	ListVideos(ctx context.Context, courseID uuid.UUID) ([]Video, error)
	ListVisibleCourses(ctx context.Context) ([]CourseListing, error)
}

type CourseWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	GetChannel(ctx context.Context, id uuid.UUID) (Channel, error)
	LockCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID) (Course, error)
	CreateCourse(ctx context.Context, tx *gorm.DB, c *Course) error
	UpdateCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID, columns map[string]any) error
	DeleteCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
	AppendChangeLog(ctx context.Context, tx *gorm.DB, rows []ChangeLog) error
}

type PlaylistWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	GetCourse(ctx context.Context, id uuid.UUID) (Course, error)
	LockCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID) (Course, error)
	ReplaceVideos(ctx context.Context, tx *gorm.DB, courseID uuid.UUID, videos []Video) error
	UpdateCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID, columns map[string]any) error
	AppendChangeLog(ctx context.Context, tx *gorm.DB, rows []ChangeLog) error
}

// ObjectStore holds uploaded channel images; *storage.R2Store satisfies it.
type ObjectStore interface {
	Put(ctx context.Context, key, contentType, cacheControl string, body []byte) error
	Delete(ctx context.Context, key string) error
}

// YouTubeChannels and YouTubePlaylists are satisfied by *youtube.Client.
type YouTubeChannels interface {
	Channel(ctx context.Context, lookup youtube.ChannelLookup) (youtube.Channel, error)
}

type YouTubePlaylists interface {
	PlaylistItems(ctx context.Context, playlistID string, maxPages int) ([]youtube.PlaylistItem, error)
	VideoDurations(ctx context.Context, videoIDs []string) (map[string]time.Duration, error)
}
