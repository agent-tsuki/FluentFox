package access

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
)

// CheckerRepo and the interfaces below are satisfied by *Repository; the user
// lookups by *users.Repository.
type CheckerRepo interface {
	IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	HasGrant(ctx context.Context, userID uuid.UUID, perm common.Permission, resources []common.ResourceRef) (bool, error)
	FindCourseChannelID(ctx context.Context, courseID uuid.UUID) (string, bool, error)
}

// Authorizer is what the admin services ask before acting; *Checker satisfies it.
type Authorizer interface {
	Has(ctx context.Context, userID uuid.UUID, perm common.Permission, res common.ResourceRef) (bool, error)
	IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

type MyPermissionsRepo interface {
	IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	ListUserGrants(ctx context.Context, userID uuid.UUID) ([]UserPermission, error)
}

type PermissionRepo interface {
	ListPermissions(ctx context.Context, email string) ([]PermissionView, error)
	CreatePermission(ctx context.Context, p *UserPermission) (UserPermission, error)
	GetPermission(ctx context.Context, id uuid.UUID) (UserPermission, error)
	DeletePermission(ctx context.Context, id uuid.UUID) error
}

type UserLookup interface {
	GetUserForEmail(ctx context.Context, email string) (users.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (users.User, error)
}

type InviteRepo interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	CreateInvite(ctx context.Context, tx *gorm.DB, inv *StaffInvite) error
	ListInvites(ctx context.Context) ([]StaffInvite, error)
	LockInvite(ctx context.Context, tx *gorm.DB, id uuid.UUID) (StaffInvite, error)
	ReplaceInviteToken(ctx context.Context, tx *gorm.DB, id uuid.UUID, tokenHash []byte, expiresAt time.Time) error
	RevokeInvite(ctx context.Context, tx *gorm.DB, id uuid.UUID, now time.Time) error
}

type InvitePreviewRepo interface {
	FindInviteByHash(ctx context.Context, tokenHash []byte) (StaffInvite, error)
	ResourceName(ctx context.Context, res common.ResourceRef) (*string, error)
}

type AcceptRepo interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	LockInviteByHash(ctx context.Context, tx *gorm.DB, tokenHash []byte) (StaffInvite, error)
	CreatePermissions(ctx context.Context, tx *gorm.DB, perms []UserPermission) error
	MarkInviteAccepted(ctx context.Context, tx *gorm.DB, id, userID uuid.UUID, now time.Time) error
}

// InviteMailer is satisfied by *services.MailService.
type InviteMailer interface {
	SendStaffInvite(ctx context.Context, to, rawToken string) error
}
