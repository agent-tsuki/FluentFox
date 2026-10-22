package classes

import (
	"time"

	"github.com/google/uuid"
)

type ListCoursesInput struct {
	Level    string `query:"level"    required:"false" enum:"N5,N4,N3,N2,N1" doc:"Only courses of this JLPT level"`
	Platform string `query:"platform" required:"false" enum:"youtube,udemy,website"`
	Free     string `query:"free"     required:"false" enum:"true,false" doc:"true for free courses only, false for paid only"`
}

type CourseSlugInput struct {
	Slug string `path:"slug" maxLength:"120"`
}

type ChannelCard struct {
	Slug            string  `json:"slug"`
	Name            string  `json:"name"`
	AvatarURL       *string `json:"avatar_url"`
	AccentColor     *string `json:"accent_color"`
	SubscriberCount *int64  `json:"subscriber_count"`
}

type CourseCard struct {
	Slug             string      `json:"slug"`
	Title            string      `json:"title"`
	Summary          string      `json:"summary"`
	Level            string      `json:"level"    enum:"N5,N4,N3,N2,N1"`
	Platform         string      `json:"platform" enum:"youtube,udemy,website"`
	IsFree           bool        `json:"is_free"`
	IsFeatured       bool        `json:"is_featured"`
	ThumbnailURL     *string     `json:"thumbnail_url"`
	VideoCount       int         `json:"video_count"        doc:"Videos in the synced playlist; 0 before the first sync"`
	TotalDurationSec int         `json:"total_duration_sec"`
	Tags             []string    `json:"tags"`
	Channel          ChannelCard `json:"channel"`
}

type LevelMeta struct {
	Level  string `json:"level"  enum:"N5,N4,N3,N2,N1"`
	Count  int    `json:"count"  doc:"Visible courses at this level, whatever the filters"`
	Locked bool   `json:"locked" doc:"True when the level has no visible course yet (coming soon)"`
}

type CatalogStats struct {
	CourseCount  int `json:"course_count"  doc:"Courses matching the filters"`
	ChannelCount int `json:"channel_count" doc:"Distinct channels of those courses"`
}

type CatalogMeta struct {
	Levels []LevelMeta  `json:"levels"`
	Stats  CatalogStats `json:"stats"`
}

type CourseListResponse struct {
	Items []CourseCard `json:"items"`
	Meta  CatalogMeta  `json:"meta"`
}

type VideoResponse struct {
	YouTubeVideoID string `json:"youtube_video_id"`
	Title          string `json:"title"`
	ThumbnailURL   string `json:"thumbnail_url"`
	DurationSec    int    `json:"duration_sec"`
	Position       int    `json:"position"`
}

type ChannelProfile struct {
	Slug             string   `json:"slug"`
	Name             string   `json:"name"`
	Platform         string   `json:"platform" enum:"youtube,udemy,website"`
	URL              string   `json:"url"`
	Handle           *string  `json:"handle"`
	YouTubeChannelID *string  `json:"youtube_channel_id"`
	AvatarURL        *string  `json:"avatar_url"`
	BannerURL        *string  `json:"banner_url"`
	Description      *string  `json:"description"`
	IntroVideoID     *string  `json:"intro_video_id"`
	SubscriberCount  *int64   `json:"subscriber_count"`
	VideoCount       *int64   `json:"video_count"`
	AccentColor      *string  `json:"accent_color"`
	TeachingStyle    string   `json:"teaching_style"`
	BestFor          []string `json:"best_for"`
	NotFor           []string `json:"not_for"`
	Pros             []string `json:"pros"`
	Cons             []string `json:"cons"`
	Verdict          string   `json:"verdict"`
}

type CourseDetail struct {
	Slug             string              `json:"slug"`
	Title            string              `json:"title"`
	Summary          string              `json:"summary"`
	Level            string              `json:"level"    enum:"N5,N4,N3,N2,N1"`
	Platform         string              `json:"platform" enum:"youtube,udemy,website"`
	URL              string              `json:"url"`
	PlaylistID       *string             `json:"playlist_id"`
	IsFree           bool                `json:"is_free"`
	PriceNote        string              `json:"price_note"`
	ThumbnailURL     *string             `json:"thumbnail_url"`
	Tags             []string            `json:"tags"`
	WhatYouLearn     []string            `json:"what_you_learn"`
	Curriculum       []CurriculumSection `json:"curriculum"`
	TotalDurationSec int                 `json:"total_duration_sec"`
	LastSyncedAt     *time.Time          `json:"last_synced_at"`
	Videos           []VideoResponse     `json:"videos"`
	Channel          ChannelProfile      `json:"channel"`
	MoreFromChannel  []CourseCard        `json:"more_from_channel"`
	Similar          []CourseCard        `json:"similar"`
}

// OverridableField shows both sources of a field and the one the public sees.
type OverridableField struct {
	YouTube   *string `json:"youtube"`
	Custom    *string `json:"custom"`
	Effective *string `json:"effective"`
}

type AdminChannelFields struct {
	Name         OverridableField `json:"name"`
	AvatarURL    OverridableField `json:"avatar_url"`
	BannerURL    OverridableField `json:"banner_url"`
	Description  OverridableField `json:"description"`
	IntroVideoID OverridableField `json:"intro_video_id"`
}

type AdminChannel struct {
	ChannelProfile
	ID           uuid.UUID          `json:"id"`
	IsPublished  bool               `json:"is_published"`
	LastSyncedAt *time.Time         `json:"last_synced_at"`
	Fields       AdminChannelFields `json:"fields"`
}

type AdminCourse struct {
	CourseDetail
	ID          uuid.UUID `json:"id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	IsPublished bool      `json:"is_published"`
	IsFeatured  bool      `json:"is_featured"`
	SortOrder   int       `json:"sort_order"`
}

type ChangeLogResponse struct {
	ID            uuid.UUID `json:"id"`
	ActorID       uuid.UUID `json:"actor_id"`
	ActorUsername *string   `json:"actor_username"`
	EntityType    string    `json:"entity_type" enum:"channel,course"`
	EntityID      uuid.UUID `json:"entity_id"`
	Field         string    `json:"field"       doc:"Column that changed, e.g. name_custom or is_published"`
	OldValue      *string   `json:"old_value"`
	NewValue      *string   `json:"new_value"`
	CreatedAt     time.Time `json:"created_at"`
}

type IDInput struct {
	ID uuid.UUID `path:"id"`
}

type IDBodyInput[T any] struct {
	ID   uuid.UUID `path:"id"`
	Body T
}

type ResetFieldInput struct {
	ID    uuid.UUID `path:"id"`
	Field string    `path:"field" enum:"name,avatar_url,banner_url,description,intro_video_id"`
}

// The bytes are the image itself (JPEG or PNG); its real type is sniffed
// from the bytes, not trusted from the header.
type ImageUploadInput struct {
	ID      uuid.UUID `path:"id"`
	RawBody []byte    `contentType:"image/jpeg"`
}

type AdminCoursesInput struct {
	ChannelID string `query:"channel_id" required:"false" format:"uuid" doc:"Only this channel's courses; all courses when omitted"`
}

type CreateChannelRequest struct {
	Slug             string  `json:"slug"                         pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"80"`
	Platform         string  `json:"platform"                     enum:"youtube,udemy,website"`
	URL              string  `json:"url"                          format:"uri" maxLength:"500"`
	Name             string  `json:"name"                         minLength:"1" maxLength:"120"`
	YouTubeChannelID *string `json:"youtube_channel_id,omitempty" maxLength:"64"`
	Handle           *string `json:"handle,omitempty"             maxLength:"100" doc:"YouTube handle, e.g. @ToKiniAndy"`
	AccentColor      *string `json:"accent_color,omitempty"       pattern:"^#[0-9A-Fa-f]{6}$"`
}

// Absent fields are left as they are; name and description set the custom
// value over the YouTube one.
type UpdateChannelRequest struct {
	Name             *string  `json:"name,omitempty"               minLength:"1" maxLength:"120"`
	Description      *string  `json:"description,omitempty"        maxLength:"5000"`
	URL              *string  `json:"url,omitempty"                format:"uri" maxLength:"500"`
	YouTubeChannelID *string  `json:"youtube_channel_id,omitempty" maxLength:"64"`
	Handle           *string  `json:"handle,omitempty"             maxLength:"100"`
	AccentColor      *string  `json:"accent_color,omitempty"       pattern:"^#[0-9A-Fa-f]{6}$"`
	TeachingStyle    *string  `json:"teaching_style,omitempty"     maxLength:"2000"`
	BestFor          []string `json:"best_for,omitempty"           maxItems:"30"`
	NotFor           []string `json:"not_for,omitempty"            maxItems:"30"`
	Pros             []string `json:"pros,omitempty"               maxItems:"30"`
	Cons             []string `json:"cons,omitempty"               maxItems:"30"`
	Verdict          *string  `json:"verdict,omitempty"            maxLength:"2000"`
}

type ChannelVisibilityRequest struct {
	IsPublished bool `json:"is_published"`
}

type IntroVideoRequest struct {
	VideoID string `json:"video_id" minLength:"1" maxLength:"300" doc:"A video id or any YouTube video URL (watch, youtu.be, embed, shorts)"`
}

type CreateCourseRequest struct {
	ChannelID    uuid.UUID           `json:"channel_id"`
	Slug         string              `json:"slug"                    pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"120"`
	Title        string              `json:"title"                   minLength:"1" maxLength:"200"`
	Level        string              `json:"level"                   enum:"N5,N4,N3,N2,N1"`
	Platform     string              `json:"platform,omitempty"      enum:"youtube,udemy,website" doc:"Defaults to the channel's platform"`
	URL          string              `json:"url"                     format:"uri" maxLength:"500"`
	PlaylistID   *string             `json:"playlist_id,omitempty"   maxLength:"64"`
	ThumbnailURL *string             `json:"thumbnail_url,omitempty" format:"uri" maxLength:"500"`
	Summary      string              `json:"summary,omitempty"       maxLength:"2000"`
	IsFree       bool                `json:"is_free,omitempty"`
	PriceNote    string              `json:"price_note,omitempty"    maxLength:"200"`
	WhatYouLearn []string            `json:"what_you_learn,omitempty" maxItems:"30"`
	Curriculum   []CurriculumSection `json:"curriculum,omitempty"     maxItems:"30"`
	Tags         []string            `json:"tags,omitempty"           maxItems:"30"`
	SortOrder    int                 `json:"sort_order,omitempty"`
}

type UpdateCourseRequest struct {
	Slug         *string             `json:"slug,omitempty"           pattern:"^[a-z0-9]+(-[a-z0-9]+)*$" maxLength:"120"`
	Title        *string             `json:"title,omitempty"          minLength:"1" maxLength:"200"`
	Level        *string             `json:"level,omitempty"          enum:"N5,N4,N3,N2,N1"`
	URL          *string             `json:"url,omitempty"            format:"uri" maxLength:"500"`
	PlaylistID   *string             `json:"playlist_id,omitempty"    maxLength:"64"`
	ThumbnailURL *string             `json:"thumbnail_url,omitempty"  maxLength:"500"`
	Summary      *string             `json:"summary,omitempty"        maxLength:"2000"`
	IsFree       *bool               `json:"is_free,omitempty"`
	PriceNote    *string             `json:"price_note,omitempty"     maxLength:"200"`
	WhatYouLearn []string            `json:"what_you_learn,omitempty" maxItems:"30"`
	Curriculum   []CurriculumSection `json:"curriculum,omitempty"     maxItems:"30"`
	Tags         []string            `json:"tags,omitempty"           maxItems:"30"`
	SortOrder    *int                `json:"sort_order,omitempty"`
}

type CourseVisibilityRequest struct {
	IsPublished bool  `json:"is_published"`
	IsFeatured  *bool `json:"is_featured,omitempty"`
}
