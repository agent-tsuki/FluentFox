// Package auth — repository.go.
// Owns all SQL for the auth domain: refresh tokens, email verifications,
// and password resets. It reads and writes plain Go types only.
// It must never contain business logic or import chi.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all auth-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs an auth Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateRefreshToken inserts a new refresh token row.
func (r *Repository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, token_hash, expires_at, created_at, revoked_at`,
		uuid.New(), userID, tokenHash, expiresAt,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt)
	if err != nil {
		return nil, fmt.Errorf("auth repository: create refresh token: %w", err)
	}
	return rt, nil
}

// GetRefreshTokenByHash fetches a refresh token by its hashed value.
func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		 FROM refresh_tokens WHERE token_hash = $1`, hash,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt)
	if err != nil {
		return nil, fmt.Errorf("auth repository: get refresh token: %w", err)
	}
	return rt, nil
}

// RevokeRefreshToken sets revoked_at to now for the given token ID.
func (r *Repository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("auth repository: revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllUserRefreshTokens revokes every active refresh token for a user.
// Called on logout-all and password change.
func (r *Repository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW()
		 WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("auth repository: revoke all user tokens: %w", err)
	}
	return nil
}

// CreateEmailVerification inserts a new email verification token.
func (r *Repository) CreateEmailVerification(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verifications (id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		uuid.New(), userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("auth repository: create email verification: %w", err)
	}
	return nil
}

// GetEmailVerificationByHash fetches an email verification row by token hash.
func (r *Repository) GetEmailVerificationByHash(ctx context.Context, hash string) (*EmailVerification, error) {
	ev := &EmailVerification{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM email_verifications WHERE token_hash = $1`, hash,
	).Scan(&ev.ID, &ev.UserID, &ev.TokenHash, &ev.ExpiresAt, &ev.UsedAt, &ev.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("auth repository: get email verification: %w", err)
	}
	return ev, nil
}

// MarkEmailVerificationUsed sets used_at for the verification row.
func (r *Repository) MarkEmailVerificationUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_verifications SET used_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("auth repository: mark verification used: %w", err)
	}
	return nil
}

// CreatePasswordReset inserts a new password reset token.
func (r *Repository) CreatePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_resets (id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		uuid.New(), userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("auth repository: create password reset: %w", err)
	}
	return nil
}

// GetPasswordResetByHash fetches a password reset row by token hash.
func (r *Repository) GetPasswordResetByHash(ctx context.Context, hash string) (*PasswordReset, error) {
	pr := &PasswordReset{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM password_resets WHERE token_hash = $1`, hash,
	).Scan(&pr.ID, &pr.UserID, &pr.TokenHash, &pr.ExpiresAt, &pr.UsedAt, &pr.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("auth repository: get password reset: %w", err)
	}
	return pr, nil
}

// MarkPasswordResetUsed sets used_at for the reset row.
func (r *Repository) MarkPasswordResetUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_resets SET used_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("auth repository: mark password reset used: %w", err)
	}
	return nil
}
