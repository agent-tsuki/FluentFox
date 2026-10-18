package profile

import (
	"time"

	"github.com/google/uuid"
)

// Avatar kinds, in the order they win: upload, preset, oauth, default.
const (
	AvatarKindUpload  = "upload"
	AvatarKindPreset  = "preset"
	AvatarKindOAuth   = "oauth"
	AvatarKindDefault = "default"
)

type AvatarResponse struct {
	Kind     string  `json:"kind"                enum:"upload,preset,oauth,default" doc:"Where the avatar comes from"`
	PresetID *string `json:"preset_id,omitempty" doc:"Preset id when kind is preset or default"`
	// A preset or default URL is the frontend's public path (e.g.
	// /avatars/men/warrior.png); upload and oauth URLs are absolute.
	URL string `json:"url" doc:"Image URL; a relative path for presets"`
}

type AvatarPresetResponse struct {
	ID    string `json:"id"`
	Group string `json:"group" enum:"male,female"`
	URL   string `json:"url"   doc:"Frontend public path of the image"`
}

type SessionLocationResponse struct {
	CountryCode *string  `json:"country_code"`
	Region      *string  `json:"region"`
	City        *string  `json:"city"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Source      *string  `json:"source"     doc:"gps is the browser's position; ip is approximate"`
	AccuracyM   *int     `json:"accuracy_m" doc:"GPS accuracy radius in metres"`
}

type SessionResponse struct {
	ID         uuid.UUID               `json:"id"`
	Browser    string                  `json:"browser"`
	OS         string                  `json:"os"`
	DeviceType string                  `json:"device_type" enum:"desktop,mobile,tablet,unknown"`
	IPApprox   string                  `json:"ip_approx"   doc:"Masked address, e.g. 203.0.113.x; empty when unknown"`
	Location   SessionLocationResponse `json:"location"`
	IsVPN      *bool                   `json:"is_vpn"      doc:"True when the login came through a VPN, so an IP location may be wrong"`
	IsCurrent  bool                    `json:"is_current"  doc:"This is the device making the request"`
	CreatedAt  time.Time               `json:"created_at"  doc:"When this device logged in"`
	LastUsedAt time.Time               `json:"last_used_at"`
}

type ProfileResponse struct {
	UserID         uuid.UUID        `json:"user_id"`
	Username       string           `json:"username"`
	Email          string           `json:"email"`
	EmailVerified  bool             `json:"email_verified"`
	FirstName      string           `json:"first_name"`
	LastName       *string          `json:"last_name"`
	Bio            *string          `json:"bio"`
	NativeLanguage string           `json:"native_language"`
	CountryCode    *string          `json:"country_code"`
	City           *string          `json:"city"`
	TargetLevel    *string          `json:"target_level" doc:"N5 to N1"`
	TimeZone       *string          `json:"time_zone"`
	DailyReminder  bool             `json:"daily_reminder"`
	ReminderTime   *string          `json:"reminder_time" doc:"HH:MM in time_zone"`
	Avatar         AvatarResponse   `json:"avatar"`
	JoinedAt       time.Time        `json:"joined_at"`
	LastLoginAt    *time.Time       `json:"last_login_at"`
	LastLogin      *SessionResponse `json:"last_login" doc:"The most recent login that is still signed in"`
}

// Every field is optional; only the ones sent change. An empty string clears
// an optional field.
type UpdateProfileRequest struct {
	FirstName      *string `json:"first_name,omitempty"      minLength:"1" maxLength:"50"`
	LastName       *string `json:"last_name,omitempty"       maxLength:"50"`
	Bio            *string `json:"bio,omitempty"             maxLength:"300"`
	NativeLanguage *string `json:"native_language,omitempty" minLength:"1" maxLength:"35"`
	CountryCode    *string `json:"country_code,omitempty"    pattern:"^([A-Z]{2})?$" doc:"ISO 3166-1 alpha-2, e.g. JP"`
	City           *string `json:"city,omitempty"            maxLength:"100"`
	TargetLevel    *string `json:"target_level,omitempty"    pattern:"^(N[1-5])?$" doc:"N5 to N1"`
	TimeZone       *string `json:"time_zone,omitempty"       maxLength:"64" doc:"IANA name, e.g. Asia/Tokyo"`
	DailyReminder  *bool   `json:"daily_reminder,omitempty"`
	ReminderTime   *string `json:"reminder_time,omitempty"   pattern:"^(([01][0-9]|2[0-3]):[0-5][0-9])?$" doc:"HH:MM"`
	AvatarPreset   *string `json:"avatar_preset,omitempty"   maxLength:"40" doc:"Pick a preset from GET /profile/avatars; replaces an uploaded image"`
}

type RevokeOthersResponse struct {
	Revoked int64 `json:"revoked" doc:"How many other devices were signed out"`
}

type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"   minimum:"-90"  maximum:"90"`
	Longitude float64 `json:"longitude"  minimum:"-180" maximum:"180"`
	AccuracyM int     `json:"accuracy_m" minimum:"0"    maximum:"100000" doc:"Accuracy radius reported by the browser"`
}
