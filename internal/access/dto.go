package access

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

type GrantRequest struct {
	Permission   common.Permission   `json:"permission" enum:"classes.channel.edit,classes.channel.media,classes.course.edit,classes.publish,classes.sync,permissions.manage"` //nolint:lll // huma reads the enum from this one tag
	ResourceType common.ResourceType `json:"resource_type,omitempty" doc:"Empty for every resource, or channel or course" maxLength:"16"`
	ResourceID   string              `json:"resource_id,omitempty"   doc:"Channel or course id; empty when resource_type is empty" maxLength:"64"`
}

func (g GrantRequest) toGrant() Grant {
	return Grant{Permission: g.Permission, ResourceType: g.ResourceType, ResourceID: strings.TrimSpace(g.ResourceID)}
}

type GrantResponse struct {
	Permission   common.Permission   `json:"permission"`
	ResourceType common.ResourceType `json:"resource_type" doc:"Empty means every resource"`
	ResourceID   string              `json:"resource_id"`
}

type MyPermissionsResponse struct {
	IsSuperAdmin bool            `json:"is_super_admin" doc:"Super-admins have every permission"`
	Grants       []GrantResponse `json:"grants"`
}

type GrantPermissionRequest struct {
	UserEmail string `json:"user_email" format:"email" maxLength:"254"`
	GrantRequest
}

type PermissionResponse struct {
	ID           uuid.UUID           `json:"id"`
	UserID       uuid.UUID           `json:"user_id"`
	UserEmail    string              `json:"user_email"`
	Username     string              `json:"username"`
	Permission   common.Permission   `json:"permission"`
	ResourceType common.ResourceType `json:"resource_type"`
	ResourceID   string              `json:"resource_id"`
	GrantedBy    *uuid.UUID          `json:"granted_by"`
	CreatedAt    time.Time           `json:"created_at"`
}

type CreateInviteRequest struct {
	Email  string         `json:"email"  format:"email" maxLength:"254"`
	Grants []GrantRequest `json:"grants" minItems:"1"   maxItems:"50"`
}

type InviteResponse struct {
	ID         uuid.UUID       `json:"id"`
	Email      string          `json:"email"`
	Status     InviteStatus    `json:"status"      enum:"pending,accepted,expired,revoked"`
	Grants     []GrantResponse `json:"grants"`
	InvitedBy  uuid.UUID       `json:"invited_by"`
	ExpiresAt  time.Time       `json:"expires_at"`
	AcceptedAt *time.Time      `json:"accepted_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

type InvitePreviewGrant struct {
	Permission   common.Permission   `json:"permission"`
	ResourceType common.ResourceType `json:"resource_type"`
	ResourceID   string              `json:"resource_id"`
	ResourceName *string             `json:"resource_name" doc:"Channel name or course title; null when unknown"`
}

type InvitePreviewResponse struct {
	EmailMasked string               `json:"email_masked" doc:"e.g. m***@gmail.com"`
	Status      InviteStatus         `json:"status"       enum:"pending,accepted,expired,revoked"`
	Grants      []InvitePreviewGrant `json:"grants"`
	ExpiresAt   time.Time            `json:"expires_at"`
}

type AcceptInviteResponse struct {
	Grants []GrantResponse `json:"grants" doc:"The grants the invite gave you"`
}

func toGrantResponse(g Grant) GrantResponse {
	return GrantResponse(g)
}

func toGrantResponses(grants []Grant) []GrantResponse {
	out := make([]GrantResponse, 0, len(grants))
	for _, g := range grants {
		out = append(out, toGrantResponse(g))
	}
	return out
}

func (p *UserPermission) grant() Grant {
	return Grant{Permission: p.Permission, ResourceType: p.ResourceType, ResourceID: p.ResourceID}
}

func toPermissionResponse(p *UserPermission, email, username string) PermissionResponse {
	return PermissionResponse{
		ID: p.ID, UserID: p.UserID, UserEmail: email, Username: username,
		Permission: p.Permission, ResourceType: p.ResourceType, ResourceID: p.ResourceID,
		GrantedBy: p.GrantedBy, CreatedAt: p.CreatedAt,
	}
}

func toInviteResponse(inv *StaffInvite, now time.Time) InviteResponse {
	return InviteResponse{
		ID: inv.ID, Email: inv.Email, Status: inv.Status(now), Grants: toGrantResponses(inv.Grants),
		InvitedBy: inv.InvitedBy, ExpiresAt: inv.ExpiresAt, AcceptedAt: inv.AcceptedAt, CreatedAt: inv.CreatedAt,
	}
}
