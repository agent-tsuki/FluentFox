package auth

import (
	"time"

	"github.com/fluentfox/api/internal/users"
	"github.com/google/uuid"
)


type RegisterRequest struct {
	Username *string  `json:"username" validate:"omitempty,min=3,max=50"`
	Email    string  `json:"email"    validate:"required,email"`
	FirstName string `json:"first_name" validate:"required,first_name"`
	LastName string `json:"last_name" validate:"required,last_name"`
	Password string  `json:"password" validate:"required,min=6"`
	PhoneNo  *string `json:"phone_no" validate:"omitempty,min=7,max=20"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UserResponse struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	Email           string    `json:"email"`
	PhoneNo         *string   `json:"phone_no"`
	IsEmailVerified bool      `json:"is_email_verified"`
	IsAdmin         bool      `json:"is_admin"`
	CreatedAt       time.Time `json:"created_at"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// toUserResponse maps a User model → UserResponse DTO
func toUserResponse(u *users.User) UserResponse {
	return UserResponse{
		ID:              u.ID,
		Username:        u.Username,
		Email:           u.Email,
		PhoneNo:         u.PhoneNo,
		IsEmailVerified: u.IsEmailVerified,
		IsAdmin:         u.IsAdmin,
		CreatedAt:       u.CreatedAt,
	}
}