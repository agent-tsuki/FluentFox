// Package user — model.go.
// DB models, request DTOs, and response DTOs for the user domain.
// The PasswordHash field appears only on the DB model — never in response DTOs.
package user

import (
	"time"

	"github.com/google/uuid"
)

// User is the DB model for the users table.
type User struct {
	ID            uuid.UUID `db:"id"`
	Email         string    `db:"email"`
	Username      string    `db:"username"`
	PasswordHash  string    `db:"password_hash"` // bcrypt hash — never sent to clients
	Role          string    `db:"role"`
	EmailVerified bool      `db:"email_verified"`
	AvatarURL     *string   `db:"avatar_url"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// Profile is the DB model for the profiles table.
type Profile struct {
	UserID      uuid.UUID `db:"user_id"`
	DisplayName *string   `db:"display_name"`
	Bio         *string   `db:"bio"`
	NativeLanguage string  `db:"native_language"`
	JLPTGoal    *string   `db:"jlpt_goal"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// UserSettings is the DB model for the user_settings table.
type UserSettings struct {
	UserID               uuid.UUID `db:"user_id"`
	DailyXPGoal          int       `db:"daily_xp_goal"`
	EmailReminders       bool      `db:"email_reminders"`
	ReminderTime         *string   `db:"reminder_time"`
	SRSCardsPerSession   int       `db:"srs_cards_per_session"`
	UpdatedAt            time.Time `db:"updated_at"`
}

// --- Request DTOs ---

// UpdateProfileRequest is the payload for PATCH /users/me/profile.
type UpdateProfileRequest struct {
	DisplayName    *string `json:"display_name"   validate:"omitempty,max=50"`
	Bio            *string `json:"bio"            validate:"omitempty,max=300"`
	NativeLanguage *string `json:"native_language" validate:"omitempty,max=50"`
	JLPTGoal       *string `json:"jlpt_goal"      validate:"omitempty,oneof=N5 N4 N3 N2 N1"`
}

// UpdateSettingsRequest is the payload for PATCH /users/me/settings.
type UpdateSettingsRequest struct {
	DailyXPGoal        *int    `json:"daily_xp_goal"          validate:"omitempty,min=10,max=500"`
	EmailReminders     *bool   `json:"email_reminders"`
	ReminderTime       *string `json:"reminder_time"          validate:"omitempty"`
	SRSCardsPerSession *int    `json:"srs_cards_per_session"  validate:"omitempty,min=5,max=100"`
}

// ChangePasswordRequest is the payload for POST /users/me/change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=8,max=72"`
}

// --- Response DTOs (no PasswordHash, ever) ---

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Username      string  `json:"username"`
	Role          string  `json:"role"`
	EmailVerified bool    `json:"email_verified"`
	AvatarURL     *string `json:"avatar_url"`
}

// ProfileResponse is the public representation of a user profile.
type ProfileResponse struct {
	DisplayName    *string `json:"display_name"`
	Bio            *string `json:"bio"`
	NativeLanguage string  `json:"native_language"`
	JLPTGoal       *string `json:"jlpt_goal"`
}

// SettingsResponse is the public representation of user settings.
type SettingsResponse struct {
	DailyXPGoal        int     `json:"daily_xp_goal"`
	EmailReminders     bool    `json:"email_reminders"`
	ReminderTime       *string `json:"reminder_time"`
	SRSCardsPerSession int     `json:"srs_cards_per_session"`
}
