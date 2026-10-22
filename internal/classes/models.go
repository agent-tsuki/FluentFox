package classes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
)

type Channel struct {
	ID                  uuid.UUID            `gorm:"type:uuid;primaryKey"`
	Slug                string               `gorm:"column:slug;not null"`
	Platform            common.ClassPlatform `gorm:"column:platform;not null"`
	URL                 string               `gorm:"column:url;not null"`
	YouTubeChannelID    *string              `gorm:"column:youtube_channel_id"`
	Handle              *string              `gorm:"column:handle"`
	NameYouTube         *string              `gorm:"column:name_youtube"`
	NameCustom          *string              `gorm:"column:name_custom"`
	AvatarURLYouTube    *string              `gorm:"column:avatar_url_youtube"`
	AvatarURLCustom     *string              `gorm:"column:avatar_url_custom"`
	BannerURLYouTube    *string              `gorm:"column:banner_url_youtube"`
	BannerURLCustom     *string              `gorm:"column:banner_url_custom"`
	DescriptionYouTube  *string              `gorm:"column:description_youtube"`
	DescriptionCustom   *string              `gorm:"column:description_custom"`
	IntroVideoIDYouTube *string              `gorm:"column:intro_video_id_youtube"`
	IntroVideoIDCustom  *string              `gorm:"column:intro_video_id_custom"`
	SubscriberCount     *int64               `gorm:"column:subscriber_count"`
	VideoCount          *int64               `gorm:"column:video_count"`
	AccentColor         *string              `gorm:"column:accent_color"`
	TeachingStyle       string               `gorm:"column:teaching_style;not null"`
	BestFor             database.TextArray   `gorm:"column:best_for;type:text[];not null"`
	NotFor              database.TextArray   `gorm:"column:not_for;type:text[];not null"`
	Pros                database.TextArray   `gorm:"column:pros;type:text[];not null"`
	Cons                database.TextArray   `gorm:"column:cons;type:text[];not null"`
	Verdict             string               `gorm:"column:verdict;not null"`
	IsPublished         bool                 `gorm:"column:is_published;not null"`
	LastSyncedAt        *time.Time           `gorm:"column:last_synced_at"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Channel) TableName() string { return "class_channels" }

func (c *Channel) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type Course struct {
	ID               uuid.UUID            `gorm:"type:uuid;primaryKey"`
	Slug             string               `gorm:"column:slug;not null"`
	ChannelID        uuid.UUID            `gorm:"column:channel_id;type:uuid;not null"`
	Title            string               `gorm:"column:title;not null"`
	Level            common.JLPTLevel     `gorm:"column:level;not null"`
	Platform         common.ClassPlatform `gorm:"column:platform;not null"`
	URL              string               `gorm:"column:url;not null"`
	PlaylistID       *string              `gorm:"column:playlist_id"`
	ThumbnailURL     *string              `gorm:"column:thumbnail_url"`
	Summary          string               `gorm:"column:summary;not null"`
	IsFree           bool                 `gorm:"column:is_free;not null"`
	PriceNote        string               `gorm:"column:price_note;not null"`
	WhatYouLearn     StringList           `gorm:"column:what_you_learn;type:jsonb;not null"`
	Curriculum       Curriculum           `gorm:"column:curriculum;type:jsonb;not null"`
	Tags             database.TextArray   `gorm:"column:tags;type:text[];not null"`
	IsPublished      bool                 `gorm:"column:is_published;not null"`
	IsFeatured       bool                 `gorm:"column:is_featured;not null"`
	SortOrder        int                  `gorm:"column:sort_order;not null"`
	TotalDurationSec int                  `gorm:"column:total_duration_sec;not null"`
	LastSyncedAt     *time.Time           `gorm:"column:last_synced_at"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Course) TableName() string { return "class_courses" }

func (c *Course) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type Video struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	CourseID       uuid.UUID `gorm:"column:course_id;type:uuid;not null"`
	YouTubeVideoID string    `gorm:"column:youtube_video_id;not null"`
	Title          string    `gorm:"column:title;not null"`
	ThumbnailURL   string    `gorm:"column:thumbnail_url;not null"`
	DurationSec    int       `gorm:"column:duration_sec;not null"`
	Position       int       `gorm:"column:position;not null"`
}

func (Video) TableName() string { return "class_videos" }

func (v *Video) BeforeCreate(_ *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

type ChangeLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActorID    uuid.UUID `gorm:"column:actor_id;type:uuid;not null"`
	EntityType string    `gorm:"column:entity_type;not null"`
	EntityID   uuid.UUID `gorm:"column:entity_id;type:uuid;not null"`
	Field      string    `gorm:"column:field;not null"`
	OldValue   *string   `gorm:"column:old_value"`
	NewValue   *string   `gorm:"column:new_value"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ChangeLog) TableName() string { return "class_change_log" }

func (l *ChangeLog) BeforeCreate(_ *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// ChangeLogEntry is a log row with the actor's username, which is nil when
// the user no longer exists.
type ChangeLogEntry struct {
	ChangeLog
	ActorUsername *string `gorm:"column:actor_username"`
}

// CourseListing is a visible course with what its card shows.
type CourseListing struct {
	Course     Course
	Channel    Channel
	VideoCount int
}

type CurriculumSection struct {
	Title  string   `json:"title"`
	Topics []string `json:"topics"`
}

// StringList maps a jsonb array of strings.
type StringList []string

func (l *StringList) Scan(src any) error { return scanJSON(src, l) }

func (l StringList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	return valueJSON(l)
}

type Curriculum []CurriculumSection

func (c *Curriculum) Scan(src any) error { return scanJSON(src, c) }

func (c Curriculum) Value() (driver.Value, error) {
	if c == nil {
		return "[]", nil
	}
	return valueJSON(c)
}

func scanJSON(src, dst any) error {
	var raw []byte
	switch v := src.(type) {
	case nil:
		return nil
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("scan jsonb: unexpected type %T", src)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("scan jsonb: %w", err)
	}
	return nil
}

func valueJSON(v any) (driver.Value, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode jsonb: %w", err)
	}
	return string(raw), nil
}
