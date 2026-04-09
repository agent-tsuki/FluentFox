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


// // 1. Start Transaction
// tx, err := r.pool.Begin(ctx)
// if err != nil {
// 	return err
// }
// // 2. Ensure rollback on failure
// defer tx.Rollback(ctx)

// // 3. Chain the operations
// userID, err := r.CreateUser(ctx, tx, email)
// if err != nil {
// 	return err
// }

// err = r.CreateProfile(ctx, tx, userID, fName, lName, lang)
// if err != nil {
// 	return err
// }

// // 4. Commit
// return tx.Commit(ctx)