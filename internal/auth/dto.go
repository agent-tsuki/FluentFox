package auth

import "github.com/google/uuid"

type RegisterRequest struct {
	Email       string  `json:"email"        validate:"required,email"   format:"email"   doc:"User's email address"`
	UserName    *string `json:"user_name,omitempty"                                          doc:"Username; auto-generated if omitted"`
	FirstName   string  `json:"first_name"   validate:"required"                             doc:"First name"`
	LastName    *string `json:"last_name,omitempty"                                          doc:"Last name"`
	PhoneNumber *string `json:"phone_number,omitempty"                                       doc:"Phone number"`
	NativeLang  string  `json:"native_lang"  validate:"required"                             doc:"Native language code (e.g. en, ja)"`
	Password    string  `json:"password"     validate:"required,min=8"   minLength:"8"       doc:"Password — minimum 8 characters"`
}


type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email" format:"email" doc:"User's email address"`
	Password string `json:"password" validate:"required,min=8" minLength:"8"  doc:"Password — minimum 8 characters"`
}

// UserSummary carries the fields a client needs immediately after auth.
type UserSummary struct {
	UserID        uuid.UUID `json:"user_id"        doc:"User's unique identifier"`
	Username      string    `json:"username"       doc:"User's handle"`
	IsAdmin       bool      `json:"is_admin"       doc:"User role: user or admin"`
	EmailVerified bool      `json:"email_verified" doc:"User verified account or not"`
	IsActive      bool      `json:"is_active"      doc:"Account is active or suspended"`
}

type LoginResponse struct {
	AccessToken  string      `json:"access_token"  doc:"Short-lived JWT access token"`
	RefreshToken string      `json:"refresh_token" doc:"Long-lived opaque refresh token"`
	User         UserSummary `json:"user"          doc:"Authenticated user summary"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" doc:"Opaque refresh token issued at login"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email" format:"email" doc:"Email address to resend the verification link to"`
}

