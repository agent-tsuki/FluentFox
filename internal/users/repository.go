package users

import (
	"context"
	"fmt"
	"time"

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

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
    return r.pool.Begin(ctx)
}

func (r *Repository) CreateUser(
	ctx context.Context, tx pgx.Tx, email string,
	username string, passwordHash string, phone_no *string,
) (uuid.UUID, error) {
	query := `
        INSERT INTO users (email, username, password_hash, phone_no) 
        VALUES ($1, $2, $3, $4) 
        RETURNING id
	`
	
	var id uuid.UUID
	err := tx.QueryRow(ctx, query, email, username, passwordHash, phone_no).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert user: %w", err)
	}
	return id, nil
}

func (r *Repository) CreateProfile(
	ctx context.Context, tx pgx.Tx, userID uuid.UUID,
	firstName string, lastName *string, nativeLang string,
) error {
	query := `
		INSERT INTO users_profile (user_id, first_name, last_name, native_language)
		VALUES ($1, $2, $3, $4)`
	
	_, err := tx.Exec(ctx, query, userID, firstName, lastName, nativeLang)
	if err != nil {
		return fmt.Errorf("failed to insert user profile: %w", err)
	}
	return nil
}

func (r *Repository) CreateUsersSettings(ctx context.Context, tx pgx.Tx, userID uuid.UUID, current_time_zone *string, reminder_time *time.Time) error {
	query := `
		INSERT INTO users_settings (user_id, current_time_zone, reminder_time)
		VALUES ($1, $2, $3)`
	
	_, err := tx.Exec(ctx, query, userID, current_time_zone, reminder_time)
	if err != nil {
		return fmt.Errorf("failed to insert user profile: %w", err)
	}
	return nil
}

func (r *Repository) CreateUserVerification(ctx context.Context, tx pgx.Tx, userID uuid.UUID, hash_code string, expires_at *time.Time) error {
	query := `
		INSERT INTO user_verification (user_id, hash_code, expires_at)
		VALUES ($1, $2, $3)`
	
	_, err := tx.Exec(ctx, query, userID, hash_code, expires_at)
	if err != nil {
		return fmt.Errorf("failed to insert user profile: %w", err)
	}
	return nil
}


func (r *Repository) GetExistingUserForEmail(ctx context.Context, email string) (bool, error) {
    query := `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`
    var exists bool
    err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
    if err != nil {
        return false, fmt.Errorf("error fetching user for email %s: %w", email, err)
    }
    return exists, nil
}


func (r *Repository) GetExistingUserForUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error fetching user for username %s: %w", username, err)
	}
	return exists, nil
}
