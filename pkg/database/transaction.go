// Package database — transaction helper.
// WithTx runs fn inside a pgx transaction. It commits on success and rolls
// back on any error or panic, then re-panics. Callers never manage BEGIN/COMMIT
// directly; they call WithTx and work inside the callback.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx executes fn within a database transaction. The pgx.Tx is passed to fn.
// If fn returns an error the transaction is rolled back and the error is returned.
// If fn panics the transaction is rolled back and the panic is re-raised.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("database: begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("database: rollback after error (%v): %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}

	return nil
}
