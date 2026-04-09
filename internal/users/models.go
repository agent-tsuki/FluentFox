package users

import (
	"time"
)

type JLPTLevel string

const (
	JLPTLevelN1 JLPTLevel = "N1"
	JLPTLevelN2 JLPTLevel = "N2"
	JLPTLevelN3 JLPTLevel = "N3"
	JLPTLevelN4 JLPTLevel = "N4"
	JLPTLevelN5 JLPTLevel = "N5"
)


type User struct {
	ID       		string    `db:"id"`
	Username 		string    `db:"username"`
	Email    		string    `db:"email"`
	PhoneNo  		*string   `db:"phone_no"`
	PasswordHash	string    `db:"password_hash"`

	// user config
	IsEmailVerified bool      `db:"is_email_verified"`
	IsAdmin         bool      `db:"is_admin"`
	IsActive        bool      `db:"is_active"`
	IsDeleted       bool      `db:"is_deleted"`

	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}


type UserProfile struct {
	ID             string     `db:"id"`
	UserID         string     `db:"user_id"`
	FirstName      string     `db:"first_name"`
	LastName        *string   `db:"last_name"`    
	Bio            *string    `db:"bio"`           
	ProfileImage   *string    `db:"profile_image"` 

	// user config
	NativeLanguage string     `db:"native_language"`
	CountryCode    *string    `db:"country_code"`  
	TargetLevel    *JLPTLevel `db:"target_level"`  

	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}


type UserSettings struct {
	ID              string    `db:"id"`
	UserID          string    `db:"user_id"`

	// location config
	CurrentTimeZone *string   `db:"current_time_zone"` 

	// config
	CursorTail          	bool      `db:"cursor_tail"`
	BackgroundAnimation 	bool      `db:"background_animation"`
	DailyReminder       	bool      `db:"daily_reminder"`
	ReminderTime        	*string   `db:"reminder_time"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}


type UserVerification struct {
	ID         string     `db:"id"`
	UserID     string     `db:"user_id"`

	// email verification
	HashCode   string     `db:"hash_code"`
	ExpiresAt  time.Time  `db:"expires_at"`
	VerifiedAt *time.Time `db:"verified_at"`  
	LastSentAt *time.Time `db:"last_sent_at"` 

	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}