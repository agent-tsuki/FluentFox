package profile

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

const cdn = "https://cdn.foxsensei.com"

func TestAvatarResolver_For(t *testing.T) {
	resolver := NewAvatarResolver(cdn + "/")
	google := "https://lh3.googleusercontent.com/a/photo"
	tests := []struct {
		name     string
		profile  users.UserProfile
		wantKind string
		wantURL  string
	}{
		{name: "upload wins over everything", profile: users.UserProfile{ProfileImage: ptr("avatars/users/u/1.jpg"), AvatarPreset: ptr("men-samurai"), OAuthImageURL: &google}, wantKind: AvatarKindUpload, wantURL: cdn + "/avatars/users/u/1.jpg"},
		{name: "picked preset wins over the Google picture", profile: users.UserProfile{AvatarPreset: ptr("women-geisha-2"), OAuthImageURL: &google}, wantKind: AvatarKindPreset, wantURL: "/avatars/women/geisha_2.png"},
		{name: "Google picture is used when nothing was picked", profile: users.UserProfile{OAuthImageURL: &google}, wantKind: AvatarKindOAuth, wantURL: google},
		{name: "unknown preset falls through to the default", profile: users.UserProfile{AvatarPreset: ptr("removed")}, wantKind: AvatarKindDefault, wantURL: "/avatars/men/warrior.png"},
		{name: "empty profile gets the default", profile: users.UserProfile{}, wantKind: AvatarKindDefault, wantURL: "/avatars/men/warrior.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.For(&tt.profile)
			if got.Kind != tt.wantKind || got.URL != tt.wantURL {
				t.Errorf("For() = %+v, want kind %q url %q", got, tt.wantKind, tt.wantURL)
			}
		})
	}
}

func TestApproximateIP(t *testing.T) {
	tests := []struct {
		name   string
		prefix *string
		want   string
	}{
		{name: "IPv4 shows the last octet as x", prefix: ptr("203.0.113.0/24"), want: "203.0.113.x"},
		{name: "IPv6 shows the masked network", prefix: ptr("2001:db8:abcd::/48"), want: "2001:db8:abcd::/48"},
		{name: "unknown is empty", prefix: nil, want: ""},
		{name: "garbage is empty", prefix: ptr("not-an-ip"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := approximateIP(tt.prefix); got != tt.want {
				t.Errorf("approximateIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProfileService_Get(t *testing.T) {
	userID := uuid.New()
	current := uuid.New()
	older := users.RefreshToken{ID: uuid.New(), CreatedAt: time.Now().Add(-time.Hour)}
	newest := users.RefreshToken{ID: current, CreatedAt: time.Now(), City: ptr("Tokyo"), UserAgent: ptr("Mozilla/5.0 (Windows NT 10.0) Chrome/130.0 Safari/537.36")}
	n3 := users.JLPTLevelN3
	base := fakeRepo{
		user:     users.User{ID: userID, Username: "fox", Email: "fox@example.com", IsEmailVerified: true},
		profile:  users.UserProfile{FirstName: "Aiko", NativeLanguage: "English", TargetLevel: &n3, City: ptr("Osaka")},
		sessions: []users.RefreshToken{older, newest},
	}
	with := func(edit func(r *fakeRepo)) *fakeRepo {
		r := base
		edit(&r)
		return &r
	}

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantErr error
	}{
		{name: "missing user is 404", repo: with(func(r *fakeRepo) { r.userErr = fmt.Errorf("x: %w", gorm.ErrRecordNotFound) }), wantErr: exceptions.ErrUserNotFound},
		{name: "missing profile is 404", repo: with(func(r *fakeRepo) { r.profileErr = fmt.Errorf("x: %w", gorm.ErrRecordNotFound) }), wantErr: exceptions.ErrProfileNotFound},
		{name: "missing settings row still returns the profile", repo: with(func(r *fakeRepo) { r.settingErr = fmt.Errorf("x: %w", gorm.ErrRecordNotFound) })},
		{name: "a settings database error is returned", repo: with(func(r *fakeRepo) { r.settingErr = errDB }), wantErr: errDB},
		{name: "maps the profile and the latest login", repo: with(func(*fakeRepo) {})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProfileService(tt.repo, NewAvatarResolver(cdn), nop()).Get(context.Background(), userID, current)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Username != "fox" || got.FirstName != "Aiko" || got.TargetLevel == nil || *got.TargetLevel != "N3" || *got.City != "Osaka" {
				t.Errorf("profile = %+v", got)
			}
			if got.LastLogin == nil || got.LastLogin.ID != current || !got.LastLogin.IsCurrent || got.LastLogin.Browser != "Chrome" {
				t.Errorf("last login = %+v, want the newest session marked current", got.LastLogin)
			}
		})
	}
}

func TestUpdateProfileService_Update(t *testing.T) {
	userID := uuid.New()
	oldKey := "avatars/users/old.jpg"
	tests := []struct {
		name        string
		repo        *fakeRepo
		req         UpdateProfileRequest
		wantErr     error
		wantDeleted []string
	}{
		{name: "bad time zone is rejected", repo: &fakeRepo{}, req: UpdateProfileRequest{TimeZone: ptr("Mars/Olympus")}, wantErr: exceptions.ErrTimeZoneInvalid},
		{name: "server-local time zone is rejected", repo: &fakeRepo{}, req: UpdateProfileRequest{TimeZone: ptr("Local")}, wantErr: exceptions.ErrTimeZoneInvalid},
		{name: "unknown preset is rejected", repo: &fakeRepo{}, req: UpdateProfileRequest{AvatarPreset: ptr("dragon")}, wantErr: exceptions.ErrAvatarPresetInvalid},
		{name: "blank first name is rejected", repo: &fakeRepo{}, req: UpdateProfileRequest{FirstName: ptr("   ")}, wantErr: exceptions.ErrValidation},
		{name: "missing profile is 404", repo: &fakeRepo{lockErr: fmt.Errorf("x: %w", gorm.ErrRecordNotFound)}, req: UpdateProfileRequest{City: ptr("Kyoto")}, wantErr: exceptions.ErrProfileNotFound},
		{name: "fields are trimmed and saved", repo: &fakeRepo{}, req: UpdateProfileRequest{City: ptr("  Kyoto "), TargetLevel: ptr("N4"), TimeZone: ptr("Asia/Tokyo")}},
		{name: "picking a preset replaces and deletes the upload", repo: &fakeRepo{profile: users.UserProfile{ProfileImage: &oldKey}}, req: UpdateProfileRequest{AvatarPreset: ptr("men-samurai")}, wantDeleted: []string{oldKey}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			avatars := NewAvatarResolver(cdn)
			reader := NewProfileService(tt.repo, avatars, nop())
			_, err := NewUpdateProfileService(tt.repo, reader, avatars, store, nop()).Update(context.Background(), userID, uuid.Nil, &tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.req.City != nil && *tt.repo.profileUpdate.City != "Kyoto" {
				t.Errorf("city = %q, want trimmed", *tt.repo.profileUpdate.City)
			}
			if fmt.Sprint(store.deleted) != fmt.Sprint(tt.wantDeleted) {
				t.Errorf("deleted = %v, want %v", store.deleted, tt.wantDeleted)
			}
			if tt.req.AvatarPreset != nil && (tt.repo.avatarPreset == nil || tt.repo.avatarKey != nil) {
				t.Errorf("avatar = preset %v key %v, want the preset and no upload", tt.repo.avatarPreset, tt.repo.avatarKey)
			}
		})
	}
}

func TestAvatarService_Upload(t *testing.T) {
	userID := uuid.New()
	oldKey := "avatars/users/old.jpg"
	tests := []struct {
		name        string
		repo        *fakeRepo
		store       *fakeStore
		noStore     bool
		body        func(t *testing.T) []byte
		wantErr     error
		wantDeleted int
	}{
		{name: "no storage configured is 503", repo: &fakeRepo{}, noStore: true, body: func(t *testing.T) []byte { return testPNG(t, 200) }, wantErr: exceptions.ErrAvatarUploadUnavailable},
		{name: "a non-image is 415", repo: &fakeRepo{}, store: &fakeStore{}, body: func(*testing.T) []byte { return []byte("GIF89a not really") }, wantErr: exceptions.ErrAvatarTypeInvalid},
		{name: "a tiny image is 422", repo: &fakeRepo{}, store: &fakeStore{}, body: func(t *testing.T) []byte { return testPNG(t, common.AVATAR_MIN_SOURCE_PX-1) }, wantErr: exceptions.ErrAvatarImageInvalid},
		{name: "a failed save deletes the new file", repo: &fakeRepo{setAvatarErr: errDB}, store: &fakeStore{}, body: func(t *testing.T) []byte { return testPNG(t, 200) }, wantErr: errDB, wantDeleted: 1},
		{name: "success stores the file and deletes the old upload", repo: &fakeRepo{profile: users.UserProfile{ProfileImage: &oldKey}}, store: &fakeStore{}, body: func(t *testing.T) []byte { return testPNG(t, 200) }, wantDeleted: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var store ObjectStore
			if !tt.noStore {
				store = tt.store
			}
			got, err := NewAvatarService(tt.repo, store, NewAvatarResolver(cdn), nop()).Upload(context.Background(), userID, tt.body(t))
			if tt.store != nil && len(tt.store.deleted) != tt.wantDeleted {
				t.Errorf("deleted = %v, want %d file(s)", tt.store.deleted, tt.wantDeleted)
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
			if got.Kind != AvatarKindUpload || len(tt.store.put) != 1 || got.URL != cdn+"/"+tt.store.put[0] || tt.store.deleted[0] != oldKey {
				t.Errorf("got %+v, put %v, deleted %v", got, tt.store.put, tt.store.deleted)
			}
		})
	}
}

func TestAvatarService_RemoveFallsBackToOAuthPicture(t *testing.T) {
	google := "https://lh3.googleusercontent.com/a/photo"
	repo := &fakeRepo{profile: users.UserProfile{AvatarPreset: ptr("men-noble"), OAuthImageURL: &google}}

	got, err := NewAvatarService(repo, &fakeStore{}, NewAvatarResolver(cdn), nop()).Remove(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != AvatarKindOAuth || repo.avatarPreset != nil || repo.avatarKey != nil {
		t.Errorf("got %+v, saved preset %v key %v", got, repo.avatarPreset, repo.avatarKey)
	}
}

func TestSessionRevokeService(t *testing.T) {
	userID, current, other := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name    string
		repo    *fakeRepo
		target  uuid.UUID
		wantErr error
	}{
		{name: "the current device cannot be revoked from the list", repo: &fakeRepo{revokeOK: true}, target: current, wantErr: exceptions.ErrCannotRevokeCurrentSession},
		{name: "another user's or an unknown session is 404", repo: &fakeRepo{revokeOK: false}, target: other, wantErr: exceptions.ErrSessionNotFound},
		{name: "another device is revoked", repo: &fakeRepo{revokeOK: true}, target: other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewSessionRevokeService(tt.repo, nop()).Revoke(context.Background(), userID, current, tt.target)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}

	if _, err := NewSessionRevokeService(&fakeRepo{}, nop()).RevokeOthers(context.Background(), userID, uuid.Nil); !errors.Is(err, exceptions.ErrTokenInvalid) {
		t.Errorf("revoke others without a sid: err = %v, want ErrTokenInvalid", err)
	}
}

func TestSessionLocationService_UpdateCurrent(t *testing.T) {
	userID, current := uuid.New(), uuid.New()
	tests := []struct {
		name     string
		session  users.RefreshToken
		wantCity string
	}{
		{name: "keeps the IP city label when not on a VPN", session: users.RefreshToken{ID: current, City: ptr("Tokyo"), IsVPN: ptr(false)}, wantCity: "Tokyo"},
		{name: "drops the IP city on a VPN, where it is likely wrong", session: users.RefreshToken{ID: current, City: ptr("Amsterdam"), IsVPN: ptr(true)}, wantCity: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{sessions: []users.RefreshToken{tt.session}, locationOK: true}
			req := UpdateLocationRequest{Latitude: 35.6894875, Longitude: 139.6917064, AccuracyM: 20}

			got, err := NewSessionLocationService(repo, nop()).UpdateCurrent(context.Background(), userID, current, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			saved := repo.savedLocation
			if saved.Source != users.LocationSourceGPS || *saved.Latitude != 35.689 || *saved.Longitude != 139.692 || saved.City != tt.wantCity {
				t.Errorf("saved = %+v, want rounded GPS with city %q", saved, tt.wantCity)
			}
			if got.Location.Source == nil || *got.Location.Source != users.LocationSourceGPS {
				t.Errorf("response location = %+v", got.Location)
			}
		})
	}

	if _, err := NewSessionLocationService(&fakeRepo{}, nop()).UpdateCurrent(context.Background(), userID, uuid.New(), UpdateLocationRequest{}); !errors.Is(err, exceptions.ErrSessionNotFound) {
		t.Errorf("unknown session: err = %v, want ErrSessionNotFound", err)
	}
}
