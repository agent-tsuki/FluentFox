package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestStaleAccountReleaser_Release(t *testing.T) {
	boom := errors.New("db down")
	conflict := exceptions.ErrEmailAlreadyInUse

	tests := []struct {
		name         string
		repo         *fakeUserRepo
		wantErr      error
		wantAnyError bool // non-AppError wrapped failure
		wantDeleted  bool
	}{
		{
			name: "returns nil when holder already gone",
			repo: &fakeUserRepo{lockErr: gorm.ErrRecordNotFound},
		},
		{
			name:    "returns conflict when holder is verified",
			repo:    &fakeUserRepo{locked: users.User{IsEmailVerified: true}},
			wantErr: conflict,
		},
		{
			name:    "returns conflict when holder has an active token",
			repo:    &fakeUserRepo{hasActive: true},
			wantErr: conflict,
		},
		{
			name:        "returns nil and deletes a stale holder",
			repo:        &fakeUserRepo{deleted: true},
			wantDeleted: true,
		},
		{
			name:        "returns conflict when delete reports no row removed",
			repo:        &fakeUserRepo{deleted: false},
			wantErr:     conflict,
			wantDeleted: true,
		},
		{
			name:         "wraps lock error",
			repo:         &fakeUserRepo{lockErr: boom},
			wantErr:      boom,
			wantAnyError: true,
		},
		{
			name:         "wraps active-token lookup error",
			repo:         &fakeUserRepo{hasActiveErr: boom},
			wantErr:      boom,
			wantAnyError: true,
		},
		{
			name:         "wraps delete error",
			repo:         &fakeUserRepo{deleteErr: boom},
			wantErr:      boom,
			wantAnyError: true,
			wantDeleted:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewStaleAccountReleaser(tc.repo, nopLogger())
			id := uuid.New()

			err := r.Release(context.Background(), nil, id, conflict)

			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantAnyError {
				if _, isApp := exceptions.As(err); isApp {
					t.Errorf("repo failure must not be an AppError: %v", err)
				}
			}
			if got := len(tc.repo.deletedIDs) > 0; got != tc.wantDeleted {
				t.Errorf("delete called = %v, want %v", got, tc.wantDeleted)
			}
			if tc.wantDeleted && tc.repo.deletedIDs[0] != id {
				t.Errorf("deleted wrong id")
			}
		})
	}
}
