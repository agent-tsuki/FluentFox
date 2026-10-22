package classes

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type CatalogFilter struct {
	Level    common.JLPTLevel
	Platform common.ClassPlatform
	// nil means free and paid.
	Free *bool
}

func (f CatalogFilter) matches(c *Course) bool {
	return (f.Level == "" || c.Level == f.Level) &&
		(f.Platform == "" || c.Platform == f.Platform) &&
		(f.Free == nil || c.IsFree == *f.Free)
}

type ListCoursesService struct {
	repo   CatalogReader
	logger *zap.Logger
}

func NewListCoursesService(repo CatalogReader, log *zap.Logger) *ListCoursesService {
	return &ListCoursesService{repo: repo, logger: log}
}

// The catalog is small, so it is filtered in memory; the level counts ignore
// the filters so a level never looks locked just because of a filter.
func (s *ListCoursesService) List(ctx context.Context, filter CatalogFilter) (CourseListResponse, error) {
	visible, err := s.repo.ListVisibleCourses(ctx)
	if err != nil {
		return CourseListResponse{}, fmt.Errorf("list classes: %w", err)
	}
	sortListings(visible)

	perLevel := map[common.JLPTLevel]int{}
	matched := make([]CourseListing, 0, len(visible))
	channels := map[string]bool{}
	for i := range visible {
		l := &visible[i]
		perLevel[l.Course.Level]++
		if filter.matches(&l.Course) {
			matched = append(matched, *l)
			channels[l.Channel.Slug] = true
		}
	}

	levels := make([]LevelMeta, 0, len(common.CLASS_LEVELS))
	for _, lvl := range common.CLASS_LEVELS {
		levels = append(levels, LevelMeta{Level: string(lvl), Count: perLevel[lvl], Locked: perLevel[lvl] == 0})
	}
	return CourseListResponse{
		Items: toCards(matched),
		Meta: CatalogMeta{
			Levels: levels,
			Stats:  CatalogStats{CourseCount: len(matched), ChannelCount: len(channels)},
		},
	}, nil
}

type CourseDetailService struct {
	repo   CatalogReader
	logger *zap.Logger
}

func NewCourseDetailService(repo CatalogReader, log *zap.Logger) *CourseDetailService {
	return &CourseDetailService{repo: repo, logger: log}
}

// A hidden course, or a course of a hidden channel, answers 404 like a
// missing one, so drafts cannot be found by guessing slugs.
func (s *CourseDetailService) Get(ctx context.Context, slug string) (CourseDetail, error) {
	course, err := s.repo.GetCourseBySlug(ctx, slug)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CourseDetail{}, exceptions.ErrClassNotFound
	}
	if err != nil {
		return CourseDetail{}, fmt.Errorf("get class: %w", err)
	}
	if !course.IsPublished {
		return CourseDetail{}, exceptions.ErrClassNotFound
	}
	ch, err := s.repo.GetChannel(ctx, course.ChannelID)
	if err != nil {
		return CourseDetail{}, fmt.Errorf("get class channel: %w", err)
	}
	if !ch.IsPublished {
		return CourseDetail{}, exceptions.ErrClassNotFound
	}
	videos, err := s.repo.ListVideos(ctx, course.ID)
	if err != nil {
		return CourseDetail{}, fmt.Errorf("get class videos: %w", err)
	}
	visible, err := s.repo.ListVisibleCourses(ctx)
	if err != nil {
		return CourseDetail{}, fmt.Errorf("get related classes: %w", err)
	}
	return toCourseDetail(&course, &ch, videos, visible), nil
}
