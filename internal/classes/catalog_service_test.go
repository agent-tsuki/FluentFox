package classes

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

func addChannel(r *fakeRepo, slug string, published bool) Channel {
	ch := Channel{ID: uuid.New(), Slug: slug, Platform: common.ClassPlatformYouTube, NameYouTube: ptr(slug + " yt"), IsPublished: published}
	r.channels[ch.ID] = ch
	return ch
}

func addCourse(r *fakeRepo, ch Channel, slug string, level common.JLPTLevel, published bool, mutate ...func(*Course)) Course {
	c := Course{ID: uuid.New(), Slug: slug, Title: slug, ChannelID: ch.ID, Level: level, Platform: ch.Platform, IsFree: true, IsPublished: published}
	for _, m := range mutate {
		m(&c)
	}
	r.courses[c.ID] = c
	return c
}

func TestEffective(t *testing.T) {
	tests := []struct {
		name    string
		custom  *string
		youtube *string
		want    *string
	}{
		{name: "custom wins over youtube", custom: ptr("Custom"), youtube: ptr("YouTube"), want: ptr("Custom")},
		{name: "youtube when no custom", youtube: ptr("YouTube"), want: ptr("YouTube")},
		{name: "empty custom still wins", custom: ptr(""), youtube: ptr("YouTube"), want: ptr("")},
		{name: "nil when neither", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effective(tt.custom, tt.youtube)
			if !equalPtr(got, tt.want) {
				t.Errorf("got %v, want %v", deref(got), deref(tt.want))
			}
		})
	}

	t.Run("admin view shows both sources and the effective value", func(t *testing.T) {
		ch := Channel{Slug: "s", NameYouTube: ptr("YT"), NameCustom: ptr("Mine"), BannerURLYouTube: ptr("yt-banner")}
		got := toAdminChannel(&ch)
		if got.Name != "Mine" || deref(got.Fields.Name.Effective) != "Mine" || deref(got.Fields.Name.YouTube) != "YT" {
			t.Errorf("name = %q, fields = %+v", got.Name, got.Fields.Name)
		}
		if deref(got.BannerURL) != "yt-banner" || got.Fields.BannerURL.Custom != nil {
			t.Errorf("banner = %v, fields = %+v", got.BannerURL, got.Fields.BannerURL)
		}
	})
}

func TestListCoursesService_List(t *testing.T) {
	repo := newFakeRepo()
	misa := addChannel(repo, "misa", true)
	dolly := addChannel(repo, "dolly", true)
	hidden := addChannel(repo, "hidden", false)
	addCourse(repo, misa, "b-course", common.JLPTLevelN5, true, func(c *Course) { c.SortOrder = 2 })
	addCourse(repo, dolly, "a-course", common.JLPTLevelN5, true, func(c *Course) { c.SortOrder = 2; c.IsFree = false })
	addCourse(repo, dolly, "featured", common.JLPTLevelN5, true, func(c *Course) { c.SortOrder = 9; c.IsFeatured = true })
	addCourse(repo, misa, "n4", common.JLPTLevelN4, true)
	addCourse(repo, misa, "draft", common.JLPTLevelN3, false)
	addCourse(repo, hidden, "hidden-channel", common.JLPTLevelN2, true)

	free := true
	tests := []struct {
		name         string
		filter       CatalogFilter
		wantSlugs    []string
		wantChannels int
	}{
		{name: "featured first then sort order then title", wantSlugs: []string{"featured", "n4", "a-course", "b-course"}, wantChannels: 2},
		{name: "level filter", filter: CatalogFilter{Level: common.JLPTLevelN4}, wantSlugs: []string{"n4"}, wantChannels: 1},
		{name: "free filter", filter: CatalogFilter{Level: common.JLPTLevelN5, Free: &free}, wantSlugs: []string{"featured", "b-course"}, wantChannels: 2},
		{name: "platform filter with no match", filter: CatalogFilter{Platform: common.ClassPlatformUdemy}, wantSlugs: []string{}, wantChannels: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewListCoursesService(repo, zap.NewNop()).List(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got.Items) != len(tt.wantSlugs) {
				t.Fatalf("got %d items, want %v", len(got.Items), tt.wantSlugs)
			}
			for i, slug := range tt.wantSlugs {
				if got.Items[i].Slug != slug {
					t.Errorf("item %d = %q, want %q", i, got.Items[i].Slug, slug)
				}
			}
			if got.Meta.Stats.CourseCount != len(tt.wantSlugs) || got.Meta.Stats.ChannelCount != tt.wantChannels {
				t.Errorf("stats = %+v", got.Meta.Stats)
			}
		})
	}

	t.Run("levels ignore filters and lock empty levels", func(t *testing.T) {
		got, err := NewListCoursesService(repo, zap.NewNop()).List(context.Background(), CatalogFilter{Level: common.JLPTLevelN4})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []LevelMeta{
			{Level: "N5", Count: 3}, {Level: "N4", Count: 1},
			{Level: "N3", Locked: true}, {Level: "N2", Locked: true}, {Level: "N1", Locked: true},
		}
		if len(got.Meta.Levels) != len(want) {
			t.Fatalf("levels = %+v", got.Meta.Levels)
		}
		for i, w := range want {
			if got.Meta.Levels[i] != w {
				t.Errorf("level %d = %+v, want %+v", i, got.Meta.Levels[i], w)
			}
		}
	})
}

func TestCourseDetailService_Get(t *testing.T) {
	repo := newFakeRepo()
	open := addChannel(repo, "open", true)
	closed := addChannel(repo, "closed", false)
	other := addChannel(repo, "other", true)
	addCourse(repo, open, "visible", common.JLPTLevelN5, true)
	addCourse(repo, open, "sibling", common.JLPTLevelN5, true)
	addCourse(repo, other, "same-level", common.JLPTLevelN5, true)
	addCourse(repo, other, "other-level", common.JLPTLevelN4, true)
	addCourse(repo, open, "draft", common.JLPTLevelN5, false)
	addCourse(repo, closed, "in-hidden-channel", common.JLPTLevelN5, true)

	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{name: "published course in a published channel", slug: "visible"},
		{name: "hidden course is not found", slug: "draft", wantErr: exceptions.ErrClassNotFound},
		{name: "course of a hidden channel is not found", slug: "in-hidden-channel", wantErr: exceptions.ErrClassNotFound},
		{name: "unknown slug is not found", slug: "nope", wantErr: exceptions.ErrClassNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCourseDetailService(repo, zap.NewNop()).Get(context.Background(), tt.slug)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Channel.Slug != "open" || got.Channel.Name != "open yt" {
				t.Errorf("channel = %+v", got.Channel)
			}
			if len(got.MoreFromChannel) != 1 || got.MoreFromChannel[0].Slug != "sibling" {
				t.Errorf("more from channel = %+v", got.MoreFromChannel)
			}
			if len(got.Similar) != 1 || got.Similar[0].Slug != "same-level" {
				t.Errorf("similar = %+v", got.Similar)
			}
			if got.Videos == nil || got.Curriculum == nil || got.Tags == nil {
				t.Error("lists must be empty arrays, not null")
			}
		})
	}
}
