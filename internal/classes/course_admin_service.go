package classes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/middleware"
)

type CourseAdminListService struct {
	repo   CourseAdminReader
	perms  PermissionChecker
	logger *zap.Logger
}

func NewCourseAdminListService(repo CourseAdminReader, perms PermissionChecker, log *zap.Logger) *CourseAdminListService {
	return &CourseAdminListService{repo: repo, perms: perms, logger: log}
}

// Without a channel the list covers every course, which needs a global grant.
func (s *CourseAdminListService) List(ctx context.Context, channelID *uuid.UUID) ([]AdminCourse, error) {
	res := common.ResourceRef{}
	if channelID != nil {
		res = channelRef(*channelID)
	}
	if err := s.perms.Require(ctx, common.PERMISSION_COURSE_EDIT, res); err != nil {
		return nil, fmt.Errorf("list admin courses: %w", err)
	}
	listings, err := s.repo.ListCourses(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("list admin courses: %w", err)
	}
	sortListings(listings)
	out := make([]AdminCourse, 0, len(listings))
	for i := range listings {
		l := &listings[i]
		out = append(out, toAdminCourse(&l.Course, &l.Channel, nil, nil))
	}
	return out, nil
}

type CourseAdminGetService struct {
	repo   CourseAdminReader
	perms  PermissionChecker
	logger *zap.Logger
}

func NewCourseAdminGetService(repo CourseAdminReader, perms PermissionChecker, log *zap.Logger) *CourseAdminGetService {
	return &CourseAdminGetService{repo: repo, perms: perms, logger: log}
}

func (s *CourseAdminGetService) Get(ctx context.Context, id uuid.UUID) (AdminCourse, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_COURSE_EDIT, courseRef(id)); err != nil {
		return AdminCourse{}, fmt.Errorf("get admin course: %w", err)
	}
	return s.load(ctx, id)
}

// load builds the admin view without a permission check, for services that
// already checked their own permission.
func (s *CourseAdminGetService) load(ctx context.Context, id uuid.UUID) (AdminCourse, error) {
	course, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return AdminCourse{}, notFoundAs(err, exceptions.ErrClassNotFound, "get admin course")
	}
	ch, err := s.repo.GetChannel(ctx, course.ChannelID)
	if err != nil {
		return AdminCourse{}, fmt.Errorf("get admin course channel: %w", err)
	}
	videos, err := s.repo.ListVideos(ctx, id)
	if err != nil {
		return AdminCourse{}, fmt.Errorf("get admin course videos: %w", err)
	}
	visible, err := s.repo.ListVisibleCourses(ctx)
	if err != nil {
		return AdminCourse{}, fmt.Errorf("get admin related courses: %w", err)
	}
	return toAdminCourse(&course, &ch, videos, visible), nil
}

type CourseCreateService struct {
	repo   CourseWriter
	reader *CourseAdminGetService
	perms  PermissionChecker
	logger *zap.Logger
}

func NewCourseCreateService(repo CourseWriter, reader *CourseAdminGetService, perms PermissionChecker, log *zap.Logger) *CourseCreateService {
	return &CourseCreateService{repo: repo, reader: reader, perms: perms, logger: log}
}

// New courses start hidden. A channel grant covers creating its courses.
func (s *CourseCreateService) Create(ctx context.Context, req *CreateCourseRequest) (AdminCourse, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_COURSE_EDIT, channelRef(req.ChannelID)); err != nil {
		return AdminCourse{}, fmt.Errorf("create course: %w", err)
	}
	if strings.TrimSpace(req.Title) == "" {
		return AdminCourse{}, exceptions.ErrClassInputInvalid
	}
	ch, err := s.repo.GetChannel(ctx, req.ChannelID)
	if err != nil {
		return AdminCourse{}, notFoundAs(err, exceptions.ErrChannelNotFound, "create course")
	}
	course := newCourse(req, ch.Platform)
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.CreateCourse(ctx, tx, &course); err != nil {
			return fmt.Errorf("insert course: %w", err)
		}
		log := ChangeLog{
			ActorID: middleware.ContextUserID(ctx), EntityType: string(common.RESOURCE_TYPE_COURSE),
			EntityID: course.ID, Field: logFieldCreated, NewValue: &course.Slug, CreatedAt: time.Now(),
		}
		if err := s.repo.AppendChangeLog(ctx, tx, []ChangeLog{log}); err != nil {
			return fmt.Errorf("log course creation: %w", err)
		}
		return nil
	})
	if err != nil {
		return AdminCourse{}, fmt.Errorf("create course: %w", err)
	}
	s.logger.Info("class course created", zap.String("slug", course.Slug), zap.Stringer("course_id", course.ID))
	return s.reader.load(ctx, course.ID)
}

func newCourse(req *CreateCourseRequest, channelPlatform common.ClassPlatform) Course {
	platform := common.ClassPlatform(req.Platform)
	if platform == "" {
		platform = channelPlatform
	}
	return Course{
		Slug: req.Slug, ChannelID: req.ChannelID, Title: strings.TrimSpace(req.Title),
		Level: common.JLPTLevel(req.Level), Platform: platform, URL: req.URL,
		PlaylistID: req.PlaylistID, ThumbnailURL: req.ThumbnailURL, Summary: req.Summary,
		IsFree: req.IsFree, PriceNote: req.PriceNote,
		WhatYouLearn: StringList(listOrEmpty(req.WhatYouLearn)),
		Curriculum:   Curriculum(req.Curriculum),
		Tags:         database.TextArray(listOrEmpty(req.Tags)),
		SortOrder:    req.SortOrder,
	}
}

type CourseUpdateService struct {
	repo   CourseWriter
	reader *CourseAdminGetService
	perms  PermissionChecker
	logger *zap.Logger
}

func NewCourseUpdateService(repo CourseWriter, reader *CourseAdminGetService, perms PermissionChecker, log *zap.Logger) *CourseUpdateService {
	return &CourseUpdateService{repo: repo, reader: reader, perms: perms, logger: log}
}

func (s *CourseUpdateService) Update(ctx context.Context, id uuid.UUID, req *UpdateCourseRequest) (AdminCourse, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_COURSE_EDIT, courseRef(id)); err != nil {
		return AdminCourse{}, fmt.Errorf("update course: %w", err)
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return AdminCourse{}, exceptions.ErrClassInputInvalid
	}
	err := updateCourse(ctx, s.repo, id, func(_ *gorm.DB, c *Course, cs *changeSet) error {
		applyCourseUpdate(c, req, cs)
		return nil
	})
	if err != nil {
		return AdminCourse{}, err
	}
	return s.reader.load(ctx, id)
}

func applyCourseUpdate(c *Course, req *UpdateCourseRequest, cs *changeSet) {
	strFields := []struct {
		column    string
		current   string
		requested *string
	}{
		{"slug", c.Slug, req.Slug},
		{"title", c.Title, req.Title},
		{"level", string(c.Level), req.Level},
		{"url", c.URL, req.URL},
		{"summary", c.Summary, req.Summary},
		{"price_note", c.PriceNote, req.PriceNote},
	}
	for _, f := range strFields {
		if f.requested != nil {
			cs.Str(f.column, f.current, *f.requested)
		}
	}
	setText(cs, "playlist_id", c.PlaylistID, req.PlaylistID)
	setText(cs, "thumbnail_url", c.ThumbnailURL, req.ThumbnailURL)
	if req.IsFree != nil {
		cs.Value("is_free", c.IsFree, *req.IsFree)
	}
	if req.SortOrder != nil {
		cs.Value("sort_order", c.SortOrder, *req.SortOrder)
	}
	if req.WhatYouLearn != nil {
		cs.Value("what_you_learn", listOrEmpty(c.WhatYouLearn), StringList(req.WhatYouLearn))
	}
	if req.Curriculum != nil {
		cs.Value("curriculum", c.Curriculum, Curriculum(req.Curriculum))
	}
	setList(cs, "tags", c.Tags, req.Tags)
}

type CourseDeleteService struct {
	repo   CourseWriter
	perms  PermissionChecker
	logger *zap.Logger
}

func NewCourseDeleteService(repo CourseWriter, perms PermissionChecker, log *zap.Logger) *CourseDeleteService {
	return &CourseDeleteService{repo: repo, perms: perms, logger: log}
}

// The change log has no foreign key, so the deletion itself stays on record.
func (s *CourseDeleteService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.perms.Require(ctx, common.PERMISSION_COURSE_EDIT, courseRef(id)); err != nil {
		return fmt.Errorf("delete course: %w", err)
	}
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		course, err := s.repo.LockCourse(ctx, tx, id)
		if err != nil {
			return notFoundAs(err, exceptions.ErrClassNotFound, "lock course")
		}
		if err := s.repo.DeleteCourse(ctx, tx, id); err != nil {
			return fmt.Errorf("delete course row: %w", err)
		}
		log := ChangeLog{
			ActorID: middleware.ContextUserID(ctx), EntityType: string(common.RESOURCE_TYPE_COURSE),
			EntityID: id, Field: logFieldDeleted, OldValue: &course.Slug, CreatedAt: time.Now(),
		}
		if err := s.repo.AppendChangeLog(ctx, tx, []ChangeLog{log}); err != nil {
			return fmt.Errorf("log course deletion: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete course: %w", err)
	}
	s.logger.Info("class course deleted", zap.Stringer("course_id", id))
	return nil
}

type CourseVisibilityService struct {
	repo   CourseWriter
	reader *CourseAdminGetService
	perms  PermissionChecker
	logger *zap.Logger
}

func NewCourseVisibilityService(repo CourseWriter, reader *CourseAdminGetService, perms PermissionChecker, log *zap.Logger) *CourseVisibilityService {
	return &CourseVisibilityService{repo: repo, reader: reader, perms: perms, logger: log}
}

func (s *CourseVisibilityService) Set(ctx context.Context, id uuid.UUID, req *CourseVisibilityRequest) (AdminCourse, error) {
	if err := s.perms.Require(ctx, common.PERMISSION_CLASSES_PUBLISH, courseRef(id)); err != nil {
		return AdminCourse{}, fmt.Errorf("set course visibility: %w", err)
	}
	err := updateCourse(ctx, s.repo, id, func(_ *gorm.DB, c *Course, cs *changeSet) error {
		cs.Value("is_published", c.IsPublished, req.IsPublished)
		if req.IsFeatured != nil {
			cs.Value("is_featured", c.IsFeatured, *req.IsFeatured)
		}
		return nil
	})
	if err != nil {
		return AdminCourse{}, err
	}
	return s.reader.load(ctx, id)
}
