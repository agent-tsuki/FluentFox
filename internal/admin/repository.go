// Package admin — repository.go.
// Owns all SQL for admin-specific operations: user management, audit log, stats.
package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles admin-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs an admin Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListUsers returns a paginated list of all users with admin fields.
func (r *Repository) ListUsers(ctx context.Context, limit, offset int) ([]*AdminUserResponse, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("admin repository: count users: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, email, username, role, email_verified,
		        COALESCE(banned, false) AS banned, created_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("admin repository: list users: %w", err)
	}
	defer rows.Close()

	var users []*AdminUserResponse
	for rows.Next() {
		u := &AdminUserResponse{}
		var id uuid.UUID
		if err := rows.Scan(&id, &u.Email, &u.Username, &u.Role,
			&u.EmailVerified, &u.Banned, &u.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("admin repository: scan user: %w", err)
		}
		u.ID = id.String()
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// BanUser sets the banned flag on a user.
func (r *Repository) BanUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET banned = true, updated_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("admin repository: ban user: %w", err)
	}
	return nil
}

// LogAudit inserts an audit log entry.
func (r *Repository) LogAudit(ctx context.Context, adminID uuid.UUID, action, resource string, resourceID *uuid.UUID, metadata string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO admin_audit_log (id, admin_id, action, resource, resource_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), adminID, action, resource, resourceID, metadata,
	)
	if err != nil {
		return fmt.Errorf("admin repository: log audit: %w", err)
	}
	return nil
}

// GetStats returns platform-level statistics for the admin dashboard.
func (r *Repository) GetStats(ctx context.Context) (*StatsResponse, error) {
	s := &StatsResponse{}
	err := r.pool.QueryRow(ctx, `
		SELECT
		    (SELECT COUNT(*) FROM users),
		    (SELECT COUNT(DISTINCT user_id) FROM srs_review_log WHERE reviewed_at >= NOW() - INTERVAL '1 day'),
		    (SELECT COUNT(*) FROM srs_review_log),
		    (SELECT COUNT(*) FROM chapters WHERE published = true)
	`).Scan(&s.TotalUsers, &s.ActiveToday, &s.TotalReviews, &s.ChaptersPublished)
	if err != nil {
		return nil, fmt.Errorf("admin repository: get stats: %w", err)
	}
	return s, nil
}
