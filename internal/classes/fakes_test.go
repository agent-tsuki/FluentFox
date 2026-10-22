package classes

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/youtube"
)

// fakePerms denies the listed permissions and allows the rest.
type fakePerms struct {
	deny  map[common.Permission]bool
	err   error
	calls []string
}

func (f *fakePerms) Require(_ context.Context, perm common.Permission, res common.ResourceRef) error {
	f.calls = append(f.calls, fmt.Sprintf("%s %s:%s", perm, res.Type, res.ID))
	if f.err != nil {
		return f.err
	}
	if f.deny[perm] {
		return exceptions.ErrForbidden
	}
	return nil
}

type fakeGrants struct {
	all bool
	ids []string
}

func (f *fakeGrants) EditableChannelIDs(context.Context) (bool, []string, error) {
	return f.all, f.ids, nil
}

// fakeRepo keeps the tables in memory. Updates are applied through the gorm
// column tags, so tests see exactly what the real UPDATE would write.
type fakeRepo struct {
	channels map[uuid.UUID]Channel
	courses  map[uuid.UUID]Course
	videos   map[uuid.UUID][]Video
	logs     []ChangeLog

	channelUpdates []map[string]any
	courseUpdates  []map[string]any
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{channels: map[uuid.UUID]Channel{}, courses: map[uuid.UUID]Course{}, videos: map[uuid.UUID][]Video{}}
}

func (f *fakeRepo) Transaction(_ context.Context, fn func(*gorm.DB) error) error { return fn(nil) }

func (f *fakeRepo) GetChannel(_ context.Context, id uuid.UUID) (Channel, error) {
	ch, ok := f.channels[id]
	if !ok {
		return Channel{}, gorm.ErrRecordNotFound
	}
	return ch, nil
}

func (f *fakeRepo) LockChannel(ctx context.Context, _ *gorm.DB, id uuid.UUID) (Channel, error) {
	return f.GetChannel(ctx, id)
}

func (f *fakeRepo) ListChannels(_ context.Context, ids []uuid.UUID) ([]Channel, error) {
	var out []Channel
	for _, ch := range f.channels {
		if ids == nil || containsID(ids, ch.ID) {
			out = append(out, ch)
		}
	}
	return out, nil
}

func (f *fakeRepo) CreateChannel(_ context.Context, _ *gorm.DB, ch *Channel) error {
	ch.ID = uuid.New()
	f.channels[ch.ID] = *ch
	return nil
}

func (f *fakeRepo) UpdateChannel(_ context.Context, _ *gorm.DB, id uuid.UUID, columns map[string]any) error {
	f.channelUpdates = append(f.channelUpdates, columns)
	ch := f.channels[id]
	if err := applyColumns(&ch, columns); err != nil {
		return err
	}
	f.channels[id] = ch
	return nil
}

func (f *fakeRepo) AppendChangeLog(_ context.Context, _ *gorm.DB, rows []ChangeLog) error {
	f.logs = append(f.logs, rows...)
	return nil
}

func (f *fakeRepo) ListChannelHistory(context.Context, uuid.UUID, int) ([]ChangeLogEntry, error) {
	return nil, nil
}

func (f *fakeRepo) GetCourse(_ context.Context, id uuid.UUID) (Course, error) {
	c, ok := f.courses[id]
	if !ok {
		return Course{}, gorm.ErrRecordNotFound
	}
	return c, nil
}

func (f *fakeRepo) GetCourseBySlug(_ context.Context, slug string) (Course, error) {
	for _, c := range f.courses {
		if c.Slug == slug {
			return c, nil
		}
	}
	return Course{}, gorm.ErrRecordNotFound
}

func (f *fakeRepo) LockCourse(ctx context.Context, _ *gorm.DB, id uuid.UUID) (Course, error) {
	return f.GetCourse(ctx, id)
}

func (f *fakeRepo) CreateCourse(_ context.Context, _ *gorm.DB, c *Course) error {
	c.ID = uuid.New()
	f.courses[c.ID] = *c
	return nil
}

func (f *fakeRepo) UpdateCourse(_ context.Context, _ *gorm.DB, id uuid.UUID, columns map[string]any) error {
	f.courseUpdates = append(f.courseUpdates, columns)
	c := f.courses[id]
	if err := applyColumns(&c, columns); err != nil {
		return err
	}
	f.courses[id] = c
	return nil
}

func (f *fakeRepo) DeleteCourse(_ context.Context, _ *gorm.DB, id uuid.UUID) error {
	delete(f.courses, id)
	delete(f.videos, id)
	return nil
}

func (f *fakeRepo) ListVideos(_ context.Context, courseID uuid.UUID) ([]Video, error) {
	return f.videos[courseID], nil
}

func (f *fakeRepo) ReplaceVideos(_ context.Context, _ *gorm.DB, courseID uuid.UUID, videos []Video) error {
	f.videos[courseID] = videos
	return nil
}

func (f *fakeRepo) ListVisibleCourses(context.Context) ([]CourseListing, error) {
	var out []CourseListing
	for _, c := range f.courses {
		ch := f.channels[c.ChannelID]
		if c.IsPublished && ch.IsPublished {
			out = append(out, CourseListing{Course: c, Channel: ch, VideoCount: len(f.videos[c.ID])})
		}
	}
	return out, nil
}

func (f *fakeRepo) ListCourses(_ context.Context, channelID *uuid.UUID) ([]CourseListing, error) {
	var out []CourseListing
	for _, c := range f.courses {
		if channelID == nil || c.ChannelID == *channelID {
			out = append(out, CourseListing{Course: c, Channel: f.channels[c.ChannelID]})
		}
	}
	return out, nil
}

func containsID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// applyColumns sets struct fields by their gorm column name.
func applyColumns(dst any, columns map[string]any) error {
	v := reflect.ValueOf(dst).Elem()
	for column, value := range columns {
		field, ok := fieldByColumn(v, column)
		if !ok {
			return fmt.Errorf("no field for column %q", column)
		}
		if err := setField(field, value); err != nil {
			return fmt.Errorf("column %q: %w", column, err)
		}
	}
	return nil
}

func fieldByColumn(v reflect.Value, column string) (reflect.Value, bool) {
	t := v.Type()
	for i := range t.NumField() {
		for _, part := range strings.Split(t.Field(i).Tag.Get("gorm"), ";") {
			if part == "column:"+column {
				return v.Field(i), true
			}
		}
	}
	return reflect.Value{}, false
}

func setField(field reflect.Value, value any) error {
	if value == nil {
		field.SetZero()
		return nil
	}
	rv := reflect.ValueOf(value)
	switch {
	case rv.Type().AssignableTo(field.Type()):
		field.Set(rv)
	case rv.Type().ConvertibleTo(field.Type()):
		field.Set(rv.Convert(field.Type()))
	case field.Kind() == reflect.Pointer && rv.Type().AssignableTo(field.Type().Elem()):
		p := reflect.New(field.Type().Elem())
		p.Elem().Set(rv)
		field.Set(p)
	default:
		return fmt.Errorf("cannot set %s from %s", field.Type(), rv.Type())
	}
	return nil
}

type fakeYouTube struct {
	channel   youtube.Channel
	items     []youtube.PlaylistItem
	durations map[string]time.Duration
	err       error

	gotLookup youtube.ChannelLookup
}

func (f *fakeYouTube) Channel(_ context.Context, lookup youtube.ChannelLookup) (youtube.Channel, error) {
	f.gotLookup = lookup
	return f.channel, f.err
}

func (f *fakeYouTube) PlaylistItems(context.Context, string, int) ([]youtube.PlaylistItem, error) {
	return f.items, f.err
}

func (f *fakeYouTube) VideoDurations(context.Context, []string) (map[string]time.Duration, error) {
	return f.durations, f.err
}

func ptr[T any](v T) *T { return &v }
