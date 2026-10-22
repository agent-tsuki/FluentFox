package youtube

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{name: "hours minutes seconds", in: "PT1H2M3S", want: time.Hour + 2*time.Minute + 3*time.Second},
		{name: "minutes only", in: "PT15M", want: 15 * time.Minute},
		{name: "seconds only", in: "PT42S", want: 42 * time.Second},
		{name: "days and hours", in: "P1DT2H", want: 26 * time.Hour},
		{name: "zero for live streams", in: "P0D", want: 0},
		{name: "zero seconds", in: "PT0S", want: 0},
		{name: "empty is invalid", in: "", wantErr: true},
		{name: "bare P is invalid", in: "P", wantErr: true},
		{name: "bare PT is invalid", in: "PT", wantErr: true},
		{name: "fractions are invalid", in: "PT1.5S", wantErr: true},
		{name: "weeks are invalid", in: "P1W", wantErr: true},
		{name: "plain number is invalid", in: "120", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.in)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidDuration) {
					t.Fatalf("err = %v, want ErrInvalidDuration", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVideoID(t *testing.T) {
	const id = "WYYaf711IEM"
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "bare id", in: id, want: id},
		{name: "bare id with spaces", in: "  " + id + " ", want: id},
		{name: "watch URL", in: "https://www.youtube.com/watch?v=" + id, want: id},
		{name: "watch URL with extra params", in: "https://www.youtube.com/watch?list=PL1&v=" + id + "&t=42s", want: id},
		{name: "mobile watch URL", in: "https://m.youtube.com/watch?v=" + id, want: id},
		{name: "watch URL without scheme", in: "youtube.com/watch?v=" + id, want: id},
		{name: "short link", in: "https://youtu.be/" + id + "?si=abc", want: id},
		{name: "embed URL", in: "https://www.youtube.com/embed/" + id, want: id},
		{name: "nocookie embed URL", in: "https://www.youtube-nocookie.com/embed/" + id, want: id},
		{name: "shorts URL", in: "https://www.youtube.com/shorts/" + id, want: id},
		{name: "live URL", in: "https://www.youtube.com/live/" + id + "?feature=share", want: id},
		{name: "too short id", in: "abc", wantErr: true},
		{name: "id with bad characters", in: "WYYaf711IE!", wantErr: true},
		{name: "other host", in: "https://vimeo.com/watch?v=" + id, wantErr: true},
		{name: "look-alike host", in: "https://youtube.com.evil.com/watch?v=" + id, wantErr: true},
		{name: "channel URL", in: "https://www.youtube.com/@ToKiniAndy", wantErr: true},
		{name: "watch URL without v", in: "https://www.youtube.com/watch?list=PL1", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVideoID(tt.in)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidVideoID) {
					t.Fatalf("got %q, err = %v, want ErrInvalidVideoID", got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

const testKey = "secret-key-123"

// newTestClient serves each API resource from a canned handler.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient(testKey, time.Second)
	c.baseURL = srv.URL + "/"
	return c
}

func TestClient_Channel(t *testing.T) {
	const body = `{"items":[{"id":"UC1","snippet":{"title":"Misa","description":"desc","customUrl":"@misa",
		"thumbnails":{"default":{"url":"d"},"high":{"url":"h"}}},
		"statistics":{"subscriberCount":"660000","videoCount":"180"},
		"brandingSettings":{"channel":{"unsubscribedTrailer":"WYYaf711IEM"},"image":{"bannerExternalUrl":"b"}}}]}`
	tests := []struct {
		name      string
		lookup    ChannelLookup
		status    int
		body      string
		wantQuery string
		wantErr   error
		wantTitle string
	}{
		{name: "by id", lookup: ChannelLookup{ID: "UC1", Handle: "@misa"}, status: 200, body: body, wantQuery: "id=UC1", wantTitle: "Misa"},
		{name: "by handle", lookup: ChannelLookup{Handle: "@misa"}, status: 200, body: body, wantQuery: "forHandle=%40misa", wantTitle: "Misa"},
		{name: "no items is not found", lookup: ChannelLookup{ID: "UC9"}, status: 200, body: `{"items":[]}`, wantQuery: "id=UC9", wantErr: ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery string
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.RawQuery
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			got, err := c.Channel(context.Background(), tt.lookup)
			if !strings.Contains(gotQuery, tt.wantQuery) || !strings.Contains(gotQuery, "key="+testKey) {
				t.Errorf("query %q, want it to contain %q and the key", gotQuery, tt.wantQuery)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Title != tt.wantTitle || got.AvatarURL != "h" || got.BannerURL != "b" || got.TrailerVideoID != "WYYaf711IEM" {
				t.Errorf("got %+v", got)
			}
			if got.SubscriberCount == nil || *got.SubscriberCount != 660000 || got.VideoCount == nil || *got.VideoCount != 180 {
				t.Errorf("counts = %v, %v", got.SubscriberCount, got.VideoCount)
			}
		})
	}
}

func TestClient_PlaylistItemsFollowsPages(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageToken") == "" {
			_, _ = w.Write([]byte(`{"nextPageToken":"p2","items":[{"snippet":{"title":"A","position":0,"resourceId":{"videoId":"aaaaaaaaaaa"}}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"snippet":{"title":"B","position":1,"resourceId":{"videoId":"bbbbbbbbbbb"}}}]}`))
	})
	tests := []struct {
		name     string
		maxPages int
		want     []string
	}{
		{name: "reads every page", maxPages: 5, want: []string{"aaaaaaaaaaa", "bbbbbbbbbbb"}},
		{name: "stops at the page cap", maxPages: 1, want: []string{"aaaaaaaaaaa"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := c.PlaylistItems(context.Background(), "PL1", tt.maxPages)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != len(tt.want) {
				t.Fatalf("got %d items, want %d", len(items), len(tt.want))
			}
			for i, id := range tt.want {
				if items[i].VideoID != id {
					t.Errorf("item %d = %q, want %q", i, items[i].VideoID, id)
				}
			}
		})
	}
}

func TestClient_VideoDurations(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"aaaaaaaaaaa","contentDetails":{"duration":"PT4M5S"}}]}`))
	})
	got, err := c.VideoDurations(context.Background(), []string{"aaaaaaaaaaa", "bbbbbbbbbbb"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["aaaaaaaaaaa"] != 4*time.Minute+5*time.Second {
		t.Errorf("duration = %v", got["aaaaaaaaaaa"])
	}
	if _, ok := got["bbbbbbbbbbb"]; ok {
		t.Error("a video the API did not return must be missing")
	}
}

func TestClient_ErrorsNeverContainTheKey(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "forbidden", status: http.StatusForbidden},
		{name: "server error", status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"error":{"message":"quota exceeded"}}`))
			})
			_, err := c.Channel(context.Background(), ChannelLookup{ID: "UC1"})
			if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
				t.Fatalf("err = %v, want the API message", err)
			}
			if strings.Contains(err.Error(), testKey) {
				t.Fatalf("error leaks the API key: %v", err)
			}
		})
	}

	t.Run("connection error", func(t *testing.T) {
		c := NewClient(testKey, time.Millisecond)
		c.baseURL = "http://127.0.0.1:1/"
		_, err := c.Channel(context.Background(), ChannelLookup{ID: "UC1"})
		if err == nil || strings.Contains(err.Error(), testKey) {
			t.Fatalf("err = %v, want an error without the key", err)
		}
	})
}
