package classes

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/youtube"
)

func TestChannelSyncService_Sync(t *testing.T) {
	remote := youtube.Channel{
		ID: "UC1", Title: "Fresh name", Description: "Fresh description", AvatarURL: "yt-avatar",
		BannerURL: "yt-banner", TrailerVideoID: "WYYaf711IEM", SubscriberCount: ptr(int64(700000)),
	}
	tests := []struct {
		name    string
		channel Channel
		yt      *fakeYouTube
		wantErr error
		check   func(t *testing.T, r *fakeRepo, got AdminChannel)
	}{
		{
			name: "custom values survive and only youtube columns are written",
			channel: Channel{
				Slug: "misa", YouTubeChannelID: ptr("UC1"), NameYouTube: ptr("Old"), NameCustom: ptr("Editor name"),
				AvatarURLCustom: ptr("custom-avatar"),
			},
			yt: &fakeYouTube{channel: remote},
			check: func(t *testing.T, r *fakeRepo, got AdminChannel) {
				if got.Name != "Editor name" || deref(got.AvatarURL) != "custom-avatar" {
					t.Errorf("effective values changed: name %q avatar %q", got.Name, deref(got.AvatarURL))
				}
				if deref(got.Fields.Name.YouTube) != "Fresh name" || deref(got.BannerURL) != "yt-banner" {
					t.Errorf("youtube values not stored: %+v", got.Fields)
				}
				for column := range r.channelUpdates[0] {
					if strings.HasSuffix(column, "_custom") {
						t.Errorf("sync wrote %s", column)
					}
				}
				if got.LastSyncedAt == nil {
					t.Error("last_synced_at not set")
				}
				if !slices.ContainsFunc(r.logs, func(l ChangeLog) bool { return l.Field == "name_youtube" }) {
					t.Errorf("change log = %+v, want a name_youtube row", r.logs)
				}
			},
		},
		{
			name:    "a channel found by handle learns its id",
			channel: Channel{Slug: "andy", Handle: ptr("@ToKiniAndy")},
			yt:      &fakeYouTube{channel: remote},
			check: func(t *testing.T, _ *fakeRepo, got AdminChannel) {
				if deref(got.YouTubeChannelID) != "UC1" {
					t.Errorf("youtube_channel_id = %v", got.YouTubeChannelID)
				}
			},
		},
		{name: "no key configured", channel: Channel{YouTubeChannelID: ptr("UC1")}, wantErr: exceptions.ErrYouTubeSyncUnavailable},
		{name: "no id or handle", channel: Channel{Slug: "web"}, yt: &fakeYouTube{}, wantErr: exceptions.ErrYouTubeSourceMissing},
		{name: "unknown channel", channel: Channel{YouTubeChannelID: ptr("UC9")}, yt: &fakeYouTube{err: youtube.ErrNotFound}, wantErr: exceptions.ErrYouTubeSourceNotFound},
		{name: "YouTube failure", channel: Channel{YouTubeChannelID: ptr("UC9")}, yt: &fakeYouTube{err: errors.New("quota")}, wantErr: exceptions.ErrYouTubeSyncFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			tt.channel.ID = uuid.New()
			repo.channels[tt.channel.ID] = tt.channel
			var yt YouTubeChannels
			if tt.yt != nil {
				yt = tt.yt
			}
			got, err := NewChannelSyncService(repo, yt, &fakePerms{}, zap.NewNop()).Sync(context.Background(), tt.channel.ID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, repo, got)
		})
	}
}

func TestPlaylistSyncService_Sync(t *testing.T) {
	repo := newFakeRepo()
	ch := addChannel(repo, "misa", true)
	course := addCourse(repo, ch, "misa-n5", common.JLPTLevelN5, true, func(c *Course) { c.PlaylistID = ptr("PL1") })
	yt := &fakeYouTube{
		items: []youtube.PlaylistItem{
			{VideoID: "aaaaaaaaaaa", Title: "One", ThumbnailURL: "thumb-1"},
			{VideoID: "privateeeee", Title: "Private video"},
			{VideoID: "bbbbbbbbbbb", Title: "Two"},
		},
		durations: map[string]time.Duration{"aaaaaaaaaaa": 90 * time.Second, "bbbbbbbbbbb": 30 * time.Second},
	}
	perms := &fakePerms{}
	reader := NewCourseAdminGetService(repo, perms, zap.NewNop())

	got, err := NewPlaylistSyncService(repo, yt, reader, perms, zap.NewNop()).Sync(context.Background(), course.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Videos) != 2 || got.Videos[1].YouTubeVideoID != "bbbbbbbbbbb" || got.Videos[1].Position != 1 {
		t.Errorf("videos = %+v, want the two public ones renumbered", got.Videos)
	}
	if got.TotalDurationSec != 120 || deref(got.ThumbnailURL) != "thumb-1" || got.LastSyncedAt == nil {
		t.Errorf("total %d thumbnail %v synced %v", got.TotalDurationSec, got.ThumbnailURL, got.LastSyncedAt)
	}
	if !slices.ContainsFunc(repo.logs, func(l ChangeLog) bool { return l.Field == "videos_synced" && deref(l.NewValue) == "2" }) {
		t.Errorf("change log = %+v", repo.logs)
	}
}

func TestIntroVideoService_Set(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "bare id", input: "TkS5APqAFHA", want: "TkS5APqAFHA"},
		{name: "watch URL", input: "https://www.youtube.com/watch?v=TkS5APqAFHA", want: "TkS5APqAFHA"},
		{name: "short link", input: "https://youtu.be/TkS5APqAFHA", want: "TkS5APqAFHA"},
		{name: "not a video", input: "https://example.com/video", wantErr: exceptions.ErrVideoIDInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			ch := addChannel(repo, "dolly", true)
			got, err := NewIntroVideoService(repo, &fakePerms{}, zap.NewNop()).Set(context.Background(), ch.ID, tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if deref(got.IntroVideoID) != tt.want || deref(got.Fields.IntroVideoID.Custom) != tt.want {
				t.Errorf("intro video = %v", got.Fields.IntroVideoID)
			}
			if len(repo.logs) != 1 || repo.logs[0].Field != "intro_video_id_custom" {
				t.Errorf("change log = %+v", repo.logs)
			}
		})
	}
}

func TestResetFieldService_Reset(t *testing.T) {
	tests := []struct {
		name     string
		field    common.ChannelField
		wantPerm common.Permission
		wantErr  error
	}{
		{name: "text field needs edit", field: common.ChannelFieldName, wantPerm: common.PERMISSION_CHANNEL_EDIT},
		{name: "media field needs media", field: common.ChannelFieldAvatarURL, wantPerm: common.PERMISSION_CHANNEL_MEDIA},
		{name: "unknown field", field: "verdict", wantErr: exceptions.ErrClassFieldInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			ch := addChannel(repo, "misa", true)
			ch.NameCustom, ch.AvatarURLCustom, ch.AvatarURLYouTube = ptr("Custom"), ptr("custom-avatar"), ptr("yt-avatar")
			repo.channels[ch.ID] = ch
			perms := &fakePerms{}
			got, err := NewResetFieldService(repo, perms, nil, "", zap.NewNop()).Reset(context.Background(), ch.ID, tt.field)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasPrefix(perms.calls[0], string(tt.wantPerm)+" ") {
				t.Errorf("checked %v, want %s", perms.calls, tt.wantPerm)
			}
			stored := repo.channels[ch.ID]
			pair, _ := stored.fieldPair(tt.field)
			if pair.Custom != nil {
				t.Errorf("custom value still set: %q", *pair.Custom)
			}
			if tt.field == common.ChannelFieldAvatarURL && deref(got.AvatarURL) != "yt-avatar" {
				t.Errorf("avatar = %v, want the YouTube one back", got.AvatarURL)
			}
		})
	}
}

func TestChannelUpdateService_LogsEachChangedField(t *testing.T) {
	repo := newFakeRepo()
	ch := addChannel(repo, "misa", true)
	req := &UpdateChannelRequest{Name: ptr("New"), Pros: []string{"Clear"}, Verdict: ptr("")}

	got, err := NewChannelUpdateService(repo, &fakePerms{}, zap.NewNop()).Update(context.Background(), ch.ID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "New" || len(got.Pros) != 1 {
		t.Errorf("got %+v", got.ChannelProfile)
	}
	fields := make([]string, 0, len(repo.logs))
	for _, l := range repo.logs {
		fields = append(fields, l.Field)
	}
	slices.Sort(fields)
	// The verdict was already empty, so it is not a change.
	if !slices.Equal(fields, []string{"name_custom", "pros"}) {
		t.Errorf("logged fields = %v", fields)
	}
}

func TestChannelListService_List(t *testing.T) {
	repo := newFakeRepo()
	a := addChannel(repo, "a", true)
	addChannel(repo, "b", false)
	tests := []struct {
		name    string
		grants  *fakeGrants
		want    int
		wantErr error
	}{
		{name: "global grant sees every channel", grants: &fakeGrants{all: true}, want: 2},
		{name: "channel grant sees that channel", grants: &fakeGrants{ids: []string{a.ID.String(), "not-a-uuid"}}, want: 1},
		{name: "no classes grant is forbidden", grants: &fakeGrants{}, wantErr: exceptions.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewChannelListService(repo, tt.grants, zap.NewNop()).List(context.Background())
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.want {
				t.Errorf("got %d channels, want %d", len(got), tt.want)
			}
		})
	}
}

// Every admin write must stop at the permission check: no row changes and
// no change-log rows.
func TestAdminServices_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		deny common.Permission
		call func(r *fakeRepo, p *fakePerms, ch Channel, c Course) error
	}{
		{name: "update channel", deny: common.PERMISSION_CHANNEL_EDIT, call: func(r *fakeRepo, p *fakePerms, ch Channel, _ Course) error {
			_, err := NewChannelUpdateService(r, p, zap.NewNop()).Update(ctx, ch.ID, &UpdateChannelRequest{Name: ptr("x")})
			return err
		}},
		{name: "create channel", deny: common.PERMISSION_CHANNEL_EDIT, call: func(r *fakeRepo, p *fakePerms, _ Channel, _ Course) error {
			_, err := NewChannelCreateService(r, p, zap.NewNop()).Create(ctx, &CreateChannelRequest{Slug: "x", Name: "x"})
			return err
		}},
		{name: "upload avatar", deny: common.PERMISSION_CHANNEL_MEDIA, call: func(r *fakeRepo, p *fakePerms, ch Channel, _ Course) error {
			_, err := NewChannelMediaService(r, p, nil, "", zap.NewNop()).UploadAvatar(ctx, ch.ID, []byte("img"))
			return err
		}},
		{name: "intro video", deny: common.PERMISSION_CHANNEL_MEDIA, call: func(r *fakeRepo, p *fakePerms, ch Channel, _ Course) error {
			_, err := NewIntroVideoService(r, p, zap.NewNop()).Set(ctx, ch.ID, "TkS5APqAFHA")
			return err
		}},
		{name: "channel visibility", deny: common.PERMISSION_CLASSES_PUBLISH, call: func(r *fakeRepo, p *fakePerms, ch Channel, _ Course) error {
			_, err := NewChannelVisibilityService(r, p, zap.NewNop()).Set(ctx, ch.ID, false)
			return err
		}},
		{name: "channel sync", deny: common.PERMISSION_CLASSES_SYNC, call: func(r *fakeRepo, p *fakePerms, ch Channel, _ Course) error {
			_, err := NewChannelSyncService(r, &fakeYouTube{}, p, zap.NewNop()).Sync(ctx, ch.ID)
			return err
		}},
		{name: "read channel needs edit or media", deny: common.PERMISSION_CHANNEL_EDIT, call: func(r *fakeRepo, p *fakePerms, ch Channel, _ Course) error {
			p.deny[common.PERMISSION_CHANNEL_MEDIA] = true
			_, err := NewChannelReadService(r, p, zap.NewNop()).Get(ctx, ch.ID)
			return err
		}},
		{name: "update course", deny: common.PERMISSION_COURSE_EDIT, call: func(r *fakeRepo, p *fakePerms, _ Channel, c Course) error {
			_, err := NewCourseUpdateService(r, NewCourseAdminGetService(r, p, zap.NewNop()), p, zap.NewNop()).Update(ctx, c.ID, &UpdateCourseRequest{Title: ptr("x")})
			return err
		}},
		{name: "delete course", deny: common.PERMISSION_COURSE_EDIT, call: func(r *fakeRepo, p *fakePerms, _ Channel, c Course) error {
			return NewCourseDeleteService(r, p, zap.NewNop()).Delete(ctx, c.ID)
		}},
		{name: "course visibility", deny: common.PERMISSION_CLASSES_PUBLISH, call: func(r *fakeRepo, p *fakePerms, _ Channel, c Course) error {
			_, err := NewCourseVisibilityService(r, NewCourseAdminGetService(r, p, zap.NewNop()), p, zap.NewNop()).Set(ctx, c.ID, &CourseVisibilityRequest{})
			return err
		}},
		{name: "playlist sync", deny: common.PERMISSION_CLASSES_SYNC, call: func(r *fakeRepo, p *fakePerms, _ Channel, c Course) error {
			_, err := NewPlaylistSyncService(r, &fakeYouTube{}, NewCourseAdminGetService(r, p, zap.NewNop()), p, zap.NewNop()).Sync(ctx, c.ID)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			ch := addChannel(repo, "misa", true)
			course := addCourse(repo, ch, "misa-n5", common.JLPTLevelN5, true, func(c *Course) { c.PlaylistID = ptr("PL1") })
			perms := &fakePerms{deny: map[common.Permission]bool{tt.deny: true}}

			err := tt.call(repo, perms, ch, course)
			if !errors.Is(err, exceptions.ErrForbidden) {
				t.Fatalf("err = %v, want ErrForbidden", err)
			}
			if len(repo.channelUpdates)+len(repo.courseUpdates)+len(repo.logs) != 0 || len(repo.courses) != 1 || len(repo.channels) != 1 {
				t.Error("a denied request changed data")
			}
		})
	}
}

func TestRegisterRoutes_RegistersEveryEndpoint(t *testing.T) {
	_, api := humatest.New(t)
	RegisterRoutes(api,
		NewCatalogHandler(nil, nil, zap.NewNop()),
		NewChannelAdminHandler(&ChannelAdminServices{}, zap.NewNop()),
		NewCourseAdminHandler(&CourseAdminServices{}, zap.NewNop()),
	)
	paths := api.OpenAPI().Paths
	tests := []struct{ path, method string }{
		{"/classes", http.MethodGet},
		{"/classes/{slug}", http.MethodGet},
		{"/admin/classes/channels", http.MethodGet},
		{"/admin/classes/channels", http.MethodPost},
		{"/admin/classes/channels/{id}", http.MethodGet},
		{"/admin/classes/channels/{id}", http.MethodPatch},
		{"/admin/classes/channels/{id}/avatar", http.MethodPost},
		{"/admin/classes/channels/{id}/banner", http.MethodPost},
		{"/admin/classes/channels/{id}/intro-video", http.MethodPatch},
		{"/admin/classes/channels/{id}/custom/{field}", http.MethodDelete},
		{"/admin/classes/channels/{id}/visibility", http.MethodPatch},
		{"/admin/classes/channels/{id}/sync", http.MethodPost},
		{"/admin/classes/channels/{id}/history", http.MethodGet},
		{"/admin/classes/courses", http.MethodGet},
		{"/admin/classes/courses", http.MethodPost},
		{"/admin/classes/courses/{id}", http.MethodGet},
		{"/admin/classes/courses/{id}", http.MethodPatch},
		{"/admin/classes/courses/{id}", http.MethodDelete},
		{"/admin/classes/courses/{id}/visibility", http.MethodPatch},
		{"/admin/classes/courses/{id}/sync-playlist", http.MethodPost},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			item := paths[tt.path]
			if item == nil {
				t.Fatalf("path %s not registered", tt.path)
			}
			ops := map[string]bool{
				http.MethodGet: item.Get != nil, http.MethodPost: item.Post != nil,
				http.MethodPatch: item.Patch != nil, http.MethodDelete: item.Delete != nil,
			}
			if !ops[tt.method] {
				t.Errorf("%s %s not registered", tt.method, tt.path)
			}
		})
	}
}
