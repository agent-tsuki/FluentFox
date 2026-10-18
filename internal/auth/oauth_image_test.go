package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type fakeOAuthImageStore struct {
	saved string
	err   error
}

func (f *fakeOAuthImageStore) SetOAuthImage(_ context.Context, _ uuid.UUID, imageURL string) error {
	f.saved = imageURL
	return f.err
}

func TestSaveOAuthImage(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "https picture is saved", raw: "https://lh3.googleusercontent.com/a/photo", want: "https://lh3.googleusercontent.com/a/photo"},
		{name: "http picture is ignored", raw: "http://example.com/a.png", want: ""},
		{name: "javascript URL is ignored", raw: "javascript:alert(1)", want: ""},
		{name: "empty picture is ignored", raw: "", want: ""},
		{name: "upper-case scheme is ignored, matching the database check", raw: "HTTPS://example.com/a.png", want: ""},
		{name: "overlong URL is ignored", raw: "https://example.com/" + strings.Repeat("a", oauthImageURLMaxLen), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeOAuthImageStore{}
			saveOAuthImage(context.Background(), store, uuid.New(), tt.raw, nopLogger())
			if store.saved != tt.want {
				t.Errorf("saved %q, want %q", store.saved, tt.want)
			}
		})
	}
}

func TestSaveOAuthImage_FailureDoesNotPanic(_ *testing.T) {
	store := &fakeOAuthImageStore{err: errors.New("db down")}
	saveOAuthImage(context.Background(), store, uuid.New(), "https://pbs.twimg.com/a_400x400.jpg", nopLogger())
}

func TestXLargeImage(t *testing.T) {
	got := xLargeImage("https://pbs.twimg.com/profile_images/1/abc_normal.jpg")
	if got != "https://pbs.twimg.com/profile_images/1/abc_400x400.jpg" {
		t.Errorf("xLargeImage() = %q", got)
	}
}
