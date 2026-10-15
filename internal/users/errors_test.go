package users

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestMapUniqueViolation(t *testing.T) {
	pg := func(code, constraint string) *pgconn.PgError {
		return &pgconn.PgError{Code: code, ConstraintName: constraint}
	}
	other := pg(common.PG_UNIQUE_VIOLATION, "idx_something_else")
	wrongCode := pg("23503", common.USERS_EMAIL_UNIQUE_INDEX)
	plain := errors.New("connection reset")

	tests := []struct {
		name          string
		in            error
		wantSentinel  *exceptions.AppError
		wantUnchanged bool
	}{
		{name: "maps the email index to ErrEmailAlreadyInUse", in: pg(common.PG_UNIQUE_VIOLATION, common.USERS_EMAIL_UNIQUE_INDEX), wantSentinel: exceptions.ErrEmailAlreadyInUse},
		{name: "maps the username index to ErrUsernameAlreadyInUse", in: pg(common.PG_UNIQUE_VIOLATION, common.USERS_USERNAME_UNIQUE_INDEX), wantSentinel: exceptions.ErrUsernameAlreadyInUse},
		{name: "maps a wrapped pg error", in: fmt.Errorf("insert user: %w", pg(common.PG_UNIQUE_VIOLATION, common.USERS_EMAIL_UNIQUE_INDEX)), wantSentinel: exceptions.ErrEmailAlreadyInUse},
		{name: "leaves other constraints unchanged", in: other, wantUnchanged: true},
		{name: "leaves other pg codes unchanged", in: wrongCode, wantUnchanged: true},
		{name: "leaves non-pg errors unchanged", in: plain, wantUnchanged: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapUniqueViolation(tc.in)

			if tc.wantUnchanged {
				if got != tc.in {
					t.Errorf("got %v, want the input error unchanged", got)
				}
				return
			}
			if !errors.Is(got, tc.wantSentinel) {
				// WithCause copies the sentinel, so compare by code instead.
				app, ok := exceptions.As(got)
				if !ok || app.Code != tc.wantSentinel.Code || app.Status != tc.wantSentinel.Status {
					t.Fatalf("got %v, want %s", got, tc.wantSentinel.Code)
				}
			}
			var pgErr *pgconn.PgError
			if !errors.As(got, &pgErr) {
				t.Error("original pg error should stay reachable as the cause")
			}
		})
	}
}
