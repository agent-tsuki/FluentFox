package users

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Username     string    `gorm:"column:username;not null"`
	Email        string    `gorm:"column:email;uniqueIndex;not null"`
	PhoneNo      *string   `gorm:"column:phone_no"`
	PasswordHash string    `gorm:"column:password_hash;not null"`

	IsEmailVerified bool `gorm:"column:is_email_verified;default:false"`
	IsAdmin         bool `gorm:"column:is_admin;default:false"`
	IsActive        bool `gorm:"column:is_active;default:true"`
	IsDeleted       bool `gorm:"column:is_deleted;default:false"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (User) TableName() string { return "users" }

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type UserProfile struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID  `gorm:"column:user_id;not null"`
	FirstName    string     `gorm:"column:first_name;not null"`
	LastName     *string    `gorm:"column:last_name"`
	Bio          *string    `gorm:"column:bio"`
	ProfileImage *string    `gorm:"column:profile_image"`
	NativeLanguage string   `gorm:"column:native_language;not null"`
	CountryCode  *string    `gorm:"column:country_code"`
	TargetLevel  *JLPTLevel `gorm:"column:target_level;type:text;check:target_level IN ('N1','N2','N3','N4','N5')"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserProfile) TableName() string { return "users_profile" }

func (u *UserProfile) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type UserSettings struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID  uuid.UUID `gorm:"column:user_id;not null"`

	CurrentTimeZone     *string `gorm:"column:current_time_zone"`
	CursorTail          bool    `gorm:"column:cursor_tail;default:false"`
	BackgroundAnimation bool    `gorm:"column:background_animation;default:false"`
	DailyReminder       bool    `gorm:"column:daily_reminder;default:false"`
	ReminderTime        *string `gorm:"column:reminder_time"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserSettings) TableName() string { return "users_settings" }

func (u *UserSettings) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type UserVerification struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID  `gorm:"column:user_id;not null"`
	HashCode   string     `gorm:"column:hash_code;not null"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	VerifiedAt *time.Time `gorm:"column:verified_at"`
	LastSentAt *time.Time `gorm:"column:last_sent_at"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserVerification) TableName() string { return "user_verification" }

func (u *UserVerification) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type RefreshToken struct {
    ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
    UserID    uuid.UUID  `gorm:"column:user_id;not null"`
    TokenHash string     `gorm:"column:token_hash;not null"`
    IsRevoked bool       `gorm:"column:is_revoked;default:false"`
    ExpiresAt time.Time  `gorm:"column:expires_at;not null"`
    CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
    UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (u *RefreshToken) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}