package users

import (
	"time"

	"github.com/google/uuid"
)

// JLPTLevel represents the JLPT proficiency level enum
type JLPTLevel string

const (
	JLPt_N1 JLPTLevel = "N1"
	JLPT_N2 JLPTLevel = "N2"
	JLPT_N3 JLPTLevel = "N3"
	JLPT_N4 JLPTLevel = "N4"
	JLPT_N5 JLPTLevel = "N5"
)

// User maps to the users table
type User struct {
	ID               uuid.UUID `db:"id"                json:"id"`
	Username         string    `db:"username"          json:"username"`
	Email            string    `db:"email"             json:"email"`
	PhoneNo          *string   `db:"phone_no"          json:"phone_no"`
	IsEmailVerified  bool      `db:"is_email_verified" json:"is_email_verified"`
	VerificationKey  *string   `db:"verification_key"  json:"-"`  // never expose
	SecretKey        string    `db:"secret_key"        json:"-"`  // never expose
	IsAdmin          bool      `db:"is_admin"          json:"is_admin"`
	IsActive         bool      `db:"is_active"         json:"is_active"`
	IsDeleted        bool      `db:"is_deleted"        json:"-"`  // never expose
	CreatedAt        time.Time `db:"created_at"        json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"        json:"updated_at"`
}

// UserProfile maps to the users_profile table
type UserProfile struct {
	ID              uuid.UUID  `db:"id"                json:"id"`
	UserID          uuid.UUID  `db:"user_id"           json:"user_id"`
	FirstName       *string    `db:"first_name"        json:"first_name"`
	LastName        *string    `db:"last_name"         json:"last_name"`
	Bio             *string    `db:"bio"               json:"bio"`
	ProfileImage    *string    `db:"profile_image"     json:"profile_image"`    // S3/R2 object key
	NativeLanguage  *string    `db:"native_language"   json:"native_language"`  // BCP-47 e.g. en, id
	CountryCode     *string    `db:"country_code"      json:"country_code"`     // ISO 3166-1 alpha-2
	TargetLevel     *JLPTLevel `db:"target_level"      json:"target_level"`
	IPAddress       *string    `db:"ip_address"        json:"ip_address"`
	Latitude        *float64   `db:"latitude"          json:"latitude"`
	Longitude       *float64   `db:"longitude"         json:"longitude"`
	CurrentTimeZone *string    `db:"current_time_zone" json:"current_time_zone"` // e.g. Asia/Tokyo
	CreatedAt       time.Time  `db:"created_at"        json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"        json:"updated_at"`
}

// UserWithProfile is used when you JOIN users + users_profile together
type UserWithProfile struct {
	User
	Profile UserProfile `db:"profile" json:"profile"`
}
