package users

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

// The database is the real guard against two signups racing for the same
// email or username; the service-level checks only give a faster answer.
func mapUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != common.PG_UNIQUE_VIOLATION {
		return err
	}

	switch pgErr.ConstraintName {
	case common.USERS_EMAIL_UNIQUE_INDEX:
		return exceptions.WithCause(exceptions.ErrEmailAlreadyInUse, err)
	case common.USERS_USERNAME_UNIQUE_INDEX:
		return exceptions.WithCause(exceptions.ErrUsernameAlreadyInUse, err)
	default:
		return err
	}
}
