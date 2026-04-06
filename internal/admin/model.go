// Package admin — model.go.
// DB models and DTOs for admin panel operations.
package admin

import (
	"time"

	"github.com/google/uuid"
)

// AdminRole is the DB model for the admin_roles table.
type AdminRole struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
}

// AdminRolePermission is a single permission entry in admin_role_permissions.
type AdminRolePermission struct {
	RoleID     uuid.UUID `db:"role_id"`
	Resource   string    `db:"resource"`
	Permission string    `db:"permission_level"` // read, write, admin
}

// AdminAuditLog is the DB model for the admin_audit_log table.
type AdminAuditLog struct {
	ID         uuid.UUID  `db:"id"`
	AdminID    uuid.UUID  `db:"admin_id"`
	Action     string     `db:"action"`
	Resource   string     `db:"resource"`
	ResourceID *uuid.UUID `db:"resource_id"`
	Metadata   string     `db:"metadata"` // JSON blob
	CreatedAt  time.Time  `db:"created_at"`
}

// --- Request DTOs ---

// BanUserRequest is the payload for POST /admin/users/{id}/ban.
type BanUserRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}

// UpdateChapterRequest allows admins to toggle chapter visibility.
type UpdateChapterRequest struct {
	Published *bool `json:"published"`
}

// --- Response DTOs ---

// AdminUserResponse extends UserResponse with admin-only fields.
type AdminUserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Role          string    `json:"role"`
	EmailVerified bool      `json:"email_verified"`
	Banned        bool      `json:"banned"`
	CreatedAt     time.Time `json:"created_at"`
}

// AuditLogResponse is the public representation of an audit log entry.
type AuditLogResponse struct {
	ID         string    `json:"id"`
	AdminID    string    `json:"admin_id"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID *string   `json:"resource_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// StatsResponse is a summary of platform statistics for the admin dashboard.
type StatsResponse struct {
	TotalUsers      int `json:"total_users"`
	ActiveToday     int `json:"active_today"`
	TotalReviews    int `json:"total_reviews"`
	ChaptersPublished int `json:"chapters_published"`
}
