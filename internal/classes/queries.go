package classes

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	if err := r.db.WithContext(ctx).Transaction(fn); err != nil {
		return fmt.Errorf("classes transaction: %w", err)
	}
	return nil
}

func (r *Repository) GetChannel(ctx context.Context, id uuid.UUID) (Channel, error) {
	var ch Channel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&ch).Error; err != nil {
		return Channel{}, fmt.Errorf("get channel %s: %w", id, err)
	}
	return ch, nil
}

func (r *Repository) ListChannels(ctx context.Context, ids []uuid.UUID) ([]Channel, error) {
	q := r.db.WithContext(ctx).Order("slug")
	if ids != nil {
		q = q.Where("id IN ?", ids)
	}
	var channels []Channel
	if err := q.Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	return channels, nil
}

func (r *Repository) GetCourse(ctx context.Context, id uuid.UUID) (Course, error) {
	var c Course
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error; err != nil {
		return Course{}, fmt.Errorf("get course %s: %w", id, err)
	}
	return c, nil
}

func (r *Repository) GetCourseBySlug(ctx context.Context, slug string) (Course, error) {
	var c Course
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&c).Error; err != nil {
		return Course{}, fmt.Errorf("get course %q: %w", slug, err)
	}
	return c, nil
}

func (r *Repository) ListVideos(ctx context.Context, courseID uuid.UUID) ([]Video, error) {
	var videos []Video
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("position").Find(&videos).Error; err != nil {
		return nil, fmt.Errorf("list videos of %s: %w", courseID, err)
	}
	return videos, nil
}

func (r *Repository) ListVisibleCourses(ctx context.Context) ([]CourseListing, error) {
	q := r.db.WithContext(ctx).
		Joins("JOIN class_channels ch ON ch.id = class_courses.channel_id").
		Where("class_courses.is_published AND ch.is_published")
	return r.listCourses(ctx, q)
}

func (r *Repository) ListCourses(ctx context.Context, channelID *uuid.UUID) ([]CourseListing, error) {
	q := r.db.WithContext(ctx)
	if channelID != nil {
		q = q.Where("class_courses.channel_id = ?", *channelID)
	}
	return r.listCourses(ctx, q)
}

// listCourses loads the courses, then their channels and video counts in two
// more queries, so the catalog costs three queries however large it is.
func (r *Repository) listCourses(ctx context.Context, q *gorm.DB) ([]CourseListing, error) {
	var courses []Course
	if err := q.Select("class_courses.*").Order("class_courses.sort_order, class_courses.title").Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	if len(courses) == 0 {
		return nil, nil
	}
	courseIDs := make([]uuid.UUID, 0, len(courses))
	channelIDs := make([]uuid.UUID, 0, len(courses))
	for i := range courses {
		courseIDs = append(courseIDs, courses[i].ID)
		channelIDs = append(channelIDs, courses[i].ChannelID)
	}
	channels, err := r.ListChannels(ctx, channelIDs)
	if err != nil {
		return nil, err
	}
	counts, err := r.videoCounts(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]Channel, len(channels))
	for i := range channels {
		byID[channels[i].ID] = channels[i]
	}
	out := make([]CourseListing, 0, len(courses))
	for i := range courses {
		c := &courses[i]
		out = append(out, CourseListing{Course: *c, Channel: byID[c.ChannelID], VideoCount: counts[c.ID]})
	}
	return out, nil
}

type videoCountRow struct {
	CourseID uuid.UUID `gorm:"column:course_id"`
	Count    int       `gorm:"column:count"`
}

func (r *Repository) videoCounts(ctx context.Context, courseIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	var rows []videoCountRow
	err := r.db.WithContext(ctx).Model(&Video{}).
		Select("course_id, count(*) AS count").
		Where("course_id IN ?", courseIDs).
		Group("course_id").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("count videos: %w", err)
	}
	out := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		out[row.CourseID] = row.Count
	}
	return out, nil
}

// The channel's own rows and those of its current courses, newest first.
func (r *Repository) ListChannelHistory(ctx context.Context, channelID uuid.UUID, limit int) ([]ChangeLogEntry, error) {
	var rows []ChangeLogEntry
	err := r.db.WithContext(ctx).Table("class_change_log AS l").
		Select("l.*, u.username AS actor_username").
		Joins("LEFT JOIN users u ON u.id = l.actor_id").
		Where("l.entity_id = ? OR l.entity_id IN (SELECT id FROM class_courses WHERE channel_id = ?)", channelID, channelID).
		Order("l.created_at DESC").Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list history of channel %s: %w", channelID, err)
	}
	return rows, nil
}

func (r *Repository) LockChannel(ctx context.Context, tx *gorm.DB, id uuid.UUID) (Channel, error) {
	var ch Channel
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", id).First(&ch).Error; err != nil {
		return Channel{}, fmt.Errorf("lock channel %s: %w", id, err)
	}
	return ch, nil
}

func (r *Repository) LockCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID) (Course, error) {
	var c Course
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", id).First(&c).Error; err != nil {
		return Course{}, fmt.Errorf("lock course %s: %w", id, err)
	}
	return c, nil
}

func (r *Repository) CreateChannel(ctx context.Context, tx *gorm.DB, ch *Channel) error {
	if err := tx.WithContext(ctx).Create(ch).Error; err != nil {
		return fmt.Errorf("create channel: %w", mapSlugErr(err))
	}
	return nil
}

func (r *Repository) UpdateChannel(ctx context.Context, tx *gorm.DB, id uuid.UUID, columns map[string]any) error {
	if err := tx.WithContext(ctx).Model(&Channel{}).Where("id = ?", id).Updates(columns).Error; err != nil {
		return fmt.Errorf("update channel %s: %w", id, err)
	}
	return nil
}

func (r *Repository) CreateCourse(ctx context.Context, tx *gorm.DB, c *Course) error {
	if err := tx.WithContext(ctx).Create(c).Error; err != nil {
		return fmt.Errorf("create course: %w", mapSlugErr(err))
	}
	return nil
}

func (r *Repository) UpdateCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID, columns map[string]any) error {
	if err := tx.WithContext(ctx).Model(&Course{}).Where("id = ?", id).Updates(columns).Error; err != nil {
		return fmt.Errorf("update course %s: %w", id, mapSlugErr(err))
	}
	return nil
}

// Videos go with the course through ON DELETE CASCADE.
func (r *Repository) DeleteCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	if err := tx.WithContext(ctx).Where("id = ?", id).Delete(&Course{}).Error; err != nil {
		return fmt.Errorf("delete course %s: %w", id, err)
	}
	return nil
}

func (r *Repository) ReplaceVideos(ctx context.Context, tx *gorm.DB, courseID uuid.UUID, videos []Video) error {
	if err := tx.WithContext(ctx).Where("course_id = ?", courseID).Delete(&Video{}).Error; err != nil {
		return fmt.Errorf("delete videos of %s: %w", courseID, err)
	}
	if len(videos) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Create(&videos).Error; err != nil {
		return fmt.Errorf("insert videos of %s: %w", courseID, err)
	}
	return nil
}

func (r *Repository) AppendChangeLog(ctx context.Context, tx *gorm.DB, rows []ChangeLog) error {
	if len(rows) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
		return fmt.Errorf("append change log: %w", err)
	}
	return nil
}

func mapSlugErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == common.PG_UNIQUE_VIOLATION {
		return exceptions.WithCause(exceptions.ErrClassSlugTaken, err)
	}
	return err
}
