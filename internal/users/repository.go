// internal/auth/repository.go
package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func UserRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ExistsByEmail checks if an email is already registered
func (r *Repository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND is_deleted = FALSE)`,
		email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("auth: check email exists: %w", err)
	}
	return exists, nil
}

// Create inserts a new user row and returns the created user
func (r *Repository) Create(ctx context.Context, user *User) (*User, error) {
	query := `
		INSERT INTO users (id, username, email, secret_key, phone_no, is_active)
		VALUES ($1, $2, $3, $4, $5, TRUE)
		RETURNING id, username, email, phone_no,
		          is_email_verified, is_admin, is_active,
		          created_at, updated_at
	`
	created := &User{}
	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Username, user.Email, user.SecretKey, user.PhoneNo,
	).Scan(
		&created.ID, &created.Username, &created.Email, &created.PhoneNo,
		&created.IsEmailVerified, &created.IsAdmin, &created.IsActive,
		&created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("auth: create user: %w", err)
	}
	return created, nil
}

// GetByEmail fetches a user by email including hashed password
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, username, email, secret_key, phone_no,
		       is_email_verified, is_admin, is_active, is_deleted,
		       created_at, updated_at
		FROM users
		WHERE email = $1 AND is_deleted = FALSE
	`
	user := &User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.SecretKey, &user.PhoneNo,
		&user.IsEmailVerified, &user.IsAdmin, &user.IsActive, &user.IsDeleted,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth: get by email: %w", err)
	}
	return user, nil
}

// GetByID fetches a user by UUID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, username, email, phone_no,
		       is_email_verified, is_admin, is_active,
		       created_at, updated_at
		FROM users
		WHERE id = $1 AND is_deleted = FALSE
	`
	user := &User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PhoneNo,
		&user.IsEmailVerified, &user.IsAdmin, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("auth: get by id: %w", err)
	}
	return user, nil
}
