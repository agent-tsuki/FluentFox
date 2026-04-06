// Package user — repository.go.
// Owns all SQL for the user domain: users, profiles, and settings tables.
// Never imports chi, never contains business logic.
package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all user-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a user Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new user row and returns the full User record.
func (r *Repository) Create(ctx context.Context, email, username, passwordHash string) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (id, email, username, password_hash, role, email_verified)
		 VALUES ($1, $2, $3, $4, 'student', false)
		 RETURNING id, email, username, password_hash, role, email_verified, avatar_url, created_at, updated_at`,
		uuid.New(), email, username, passwordHash,
	).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &u.EmailVerified, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user repository: create user: %w", err)
	}
	return u, nil
}

// GetByID fetches a user by primary key.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, username, password_hash, role, email_verified, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &u.EmailVerified, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user repository: get by id: %w", err)
	}
	return u, nil
}

// GetByEmail fetches a user by email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, username, password_hash, role, email_verified, avatar_url, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash,
		&u.Role, &u.EmailVerified, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user repository: get by email: %w", err)
	}
	return u, nil
}

// SetEmailVerified marks a user's email as verified.
func (r *Repository) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email_verified = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("user repository: set email verified: %w", err)
	}
	return nil
}

// UpdatePasswordHash replaces a user's password hash.
func (r *Repository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, hash, id)
	if err != nil {
		return fmt.Errorf("user repository: update password hash: %w", err)
	}
	return nil
}

// UpdateAvatarURL sets the user's avatar_url.
func (r *Repository) UpdateAvatarURL(ctx context.Context, id uuid.UUID, url string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE id = $2`, url, id)
	if err != nil {
		return fmt.Errorf("user repository: update avatar url: %w", err)
	}
	return nil
}

// GetProfile fetches the profile row for the given user.
func (r *Repository) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	p := &Profile{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, display_name, bio, native_language, jlpt_goal, updated_at
		 FROM profiles WHERE user_id = $1`, userID,
	).Scan(&p.UserID, &p.DisplayName, &p.Bio, &p.NativeLanguage, &p.JLPTGoal, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user repository: get profile: %w", err)
	}
	return p, nil
}

// UpsertProfile inserts or updates the profile for the given user.
func (r *Repository) UpsertProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO profiles (user_id, display_name, bio, native_language, jlpt_goal, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (user_id) DO UPDATE
		 SET display_name = EXCLUDED.display_name,
		     bio = EXCLUDED.bio,
		     native_language = COALESCE(EXCLUDED.native_language, profiles.native_language),
		     jlpt_goal = EXCLUDED.jlpt_goal,
		     updated_at = NOW()`,
		userID, req.DisplayName, req.Bio, req.NativeLanguage, req.JLPTGoal,
	)
	if err != nil {
		return fmt.Errorf("user repository: upsert profile: %w", err)
	}
	return nil
}

// GetSettings fetches the settings row for the given user.
func (r *Repository) GetSettings(ctx context.Context, userID uuid.UUID) (*UserSettings, error) {
	s := &UserSettings{}
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, daily_xp_goal, email_reminders, reminder_time, srs_cards_per_session, updated_at
		 FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&s.UserID, &s.DailyXPGoal, &s.EmailReminders, &s.ReminderTime, &s.SRSCardsPerSession, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user repository: get settings: %w", err)
	}
	return s, nil
}

// UpsertSettings inserts or updates user settings.
func (r *Repository) UpsertSettings(ctx context.Context, userID uuid.UUID, req UpdateSettingsRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, daily_xp_goal, email_reminders, reminder_time, srs_cards_per_session, updated_at)
		 VALUES ($1, COALESCE($2, 50), COALESCE($3, true), $4, COALESCE($5, 20), NOW())
		 ON CONFLICT (user_id) DO UPDATE
		 SET daily_xp_goal = COALESCE(EXCLUDED.daily_xp_goal, user_settings.daily_xp_goal),
		     email_reminders = COALESCE(EXCLUDED.email_reminders, user_settings.email_reminders),
		     reminder_time = COALESCE(EXCLUDED.reminder_time, user_settings.reminder_time),
		     srs_cards_per_session = COALESCE(EXCLUDED.srs_cards_per_session, user_settings.srs_cards_per_session),
		     updated_at = NOW()`,
		userID, req.DailyXPGoal, req.EmailReminders, req.ReminderTime, req.SRSCardsPerSession,
	)
	if err != nil {
		return fmt.Errorf("user repository: upsert settings: %w", err)
	}
	return nil
}

// DeleteByID hard-deletes a user (GDPR erasure). Must cascade in DB.
func (r *Repository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("user repository: delete user: %w", err)
	}
	return nil
}

// ListForAdmin returns a page of users for admin use.
func (r *Repository) ListForAdmin(ctx context.Context, limit, offset int) ([]*User, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("user repository: count users: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, email, username, password_hash, role, email_verified, avatar_url, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("user repository: list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash,
			&u.Role, &u.EmailVerified, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("user repository: scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// UpdatedAt satisfies the interface for password change timestamp.
func (r *Repository) TouchUpdatedAt(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}
