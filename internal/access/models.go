package access

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type UserPermission struct {
	ID           uuid.UUID           `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID           `gorm:"column:user_id;type:uuid;not null"`
	Permission   common.Permission   `gorm:"column:permission;not null"`
	ResourceType common.ResourceType `gorm:"column:resource_type;not null"`
	ResourceID   string              `gorm:"column:resource_id;not null"`
	GrantedBy    *uuid.UUID          `gorm:"column:granted_by;type:uuid"`
	CreatedAt    time.Time           `gorm:"column:created_at;autoCreateTime"`
}

func (UserPermission) TableName() string { return "user_permissions" }

func (p *UserPermission) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PermissionView is a grant joined with the user it belongs to.
type PermissionView struct {
	UserPermission
	UserEmail string `gorm:"column:user_email"`
	Username  string `gorm:"column:username"`
}

type Grant struct {
	Permission   common.Permission   `json:"permission"`
	ResourceType common.ResourceType `json:"resource_type"`
	ResourceID   string              `json:"resource_id"`
}

func (g Grant) Resource() common.ResourceRef {
	return common.ResourceRef{Type: g.ResourceType, ID: g.ResourceID}
}

// Grants is stored as a jsonb array on staff_invites.
type Grants []Grant

func (g Grants) Value() (driver.Value, error) {
	if g == nil {
		g = Grants{}
	}
	b, err := json.Marshal(g)
	if err != nil {
		return nil, fmt.Errorf("encode grants: %w", err)
	}
	return string(b), nil
}

func (g *Grants) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	case nil:
		*g = Grants{}
		return nil
	default:
		return errors.New("grants: unsupported column type")
	}
	if err := json.Unmarshal(raw, g); err != nil {
		return fmt.Errorf("decode grants: %w", err)
	}
	return nil
}

type StaffInvite struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email      string     `gorm:"column:email;not null"`
	TokenHash  []byte     `gorm:"column:token_hash;type:bytea;not null"`
	InvitedBy  uuid.UUID  `gorm:"column:invited_by;type:uuid;not null"`
	Grants     Grants     `gorm:"column:grants;type:jsonb;not null"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	AcceptedAt *time.Time `gorm:"column:accepted_at"`
	AcceptedBy *uuid.UUID `gorm:"column:accepted_by;type:uuid"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (StaffInvite) TableName() string { return "staff_invites" }

func (i *StaffInvite) BeforeCreate(_ *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// Revoked and accepted win over expired: they are what an admin did, while
// expiry only says nobody acted in time.
func (i *StaffInvite) Status(now time.Time) InviteStatus {
	switch {
	case i.RevokedAt != nil:
		return InviteStatusRevoked
	case i.AcceptedAt != nil:
		return InviteStatusAccepted
	case !now.Before(i.ExpiresAt):
		return InviteStatusExpired
	default:
		return InviteStatusPending
	}
}

// Maps a status that blocks accepting to its client error.
func (i *StaffInvite) acceptErr(now time.Time) error {
	switch i.Status(now) {
	case InviteStatusRevoked:
		return exceptions.ErrInviteRevoked
	case InviteStatusAccepted:
		return exceptions.ErrInviteUsed
	case InviteStatusExpired:
		return exceptions.ErrInviteExpired
	default:
		return nil
	}
}
