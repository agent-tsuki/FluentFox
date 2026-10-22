package access

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type MyPermissionsService struct {
	repo   MyPermissionsRepo
	logger *zap.Logger
}

func NewMyPermissionsService(repo MyPermissionsRepo, log *zap.Logger) *MyPermissionsService {
	return &MyPermissionsService{repo: repo, logger: log}
}

func (s *MyPermissionsService) Get(ctx context.Context, userID uuid.UUID) (MyPermissionsResponse, error) {
	admin, err := s.repo.IsSuperAdmin(ctx, userID)
	if err != nil {
		return MyPermissionsResponse{}, fmt.Errorf("my permissions: %w", err)
	}
	rows, err := s.repo.ListUserGrants(ctx, userID)
	if err != nil {
		return MyPermissionsResponse{}, fmt.Errorf("my permissions: %w", err)
	}
	grants := make([]GrantResponse, 0, len(rows))
	for i := range rows {
		grants = append(grants, toGrantResponse(rows[i].grant()))
	}
	return MyPermissionsResponse{IsSuperAdmin: admin, Grants: grants}, nil
}

type ListPermissionsService struct {
	authz  Authorizer
	repo   PermissionRepo
	logger *zap.Logger
}

func NewListPermissionsService(authz Authorizer, repo PermissionRepo, log *zap.Logger) *ListPermissionsService {
	return &ListPermissionsService{authz: authz, repo: repo, logger: log}
}

func (s *ListPermissionsService) List(ctx context.Context, actorID uuid.UUID, userEmail string) ([]PermissionResponse, error) {
	if err := requireManager(ctx, s.authz, actorID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListPermissions(ctx, normalizeEmail(userEmail))
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	out := make([]PermissionResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toPermissionResponse(&rows[i].UserPermission, rows[i].UserEmail, rows[i].Username))
	}
	return out, nil
}

type GrantPermissionService struct {
	authz  Authorizer
	repo   PermissionRepo
	users  UserLookup
	logger *zap.Logger
}

func NewGrantPermissionService(authz Authorizer, repo PermissionRepo, users UserLookup, log *zap.Logger) *GrantPermissionService {
	return &GrantPermissionService{authz: authz, repo: repo, users: users, logger: log}
}

func (s *GrantPermissionService) Grant(ctx context.Context, actorID uuid.UUID, req *GrantPermissionRequest) (PermissionResponse, error) {
	grant := req.toGrant()
	if err := authorizeGrants(ctx, s.authz, actorID, []Grant{grant}); err != nil {
		return PermissionResponse{}, err
	}
	user, err := s.users.GetUserForEmail(ctx, normalizeEmail(req.UserEmail))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PermissionResponse{}, exceptions.WithCause(exceptions.ErrUserEmailNotFound, err)
	}
	if err != nil {
		return PermissionResponse{}, fmt.Errorf("grant permission: %w", err)
	}
	if user.ID == actorID {
		return PermissionResponse{}, exceptions.ErrPermissionSelfGrant
	}
	stored, err := s.repo.CreatePermission(ctx, &UserPermission{
		UserID: user.ID, Permission: grant.Permission, ResourceType: grant.ResourceType,
		ResourceID: grant.ResourceID, GrantedBy: &actorID,
	})
	if err != nil {
		return PermissionResponse{}, fmt.Errorf("grant permission: %w", err)
	}
	s.logger.Info("permission granted",
		zap.Stringer("actor_id", actorID), zap.Stringer("user_id", user.ID),
		zap.String("permission", string(grant.Permission)),
		zap.String("resource_type", string(grant.ResourceType)), zap.String("resource_id", grant.ResourceID))
	return toPermissionResponse(&stored, user.Email, user.Username), nil
}

type RevokePermissionService struct {
	authz  Authorizer
	repo   PermissionRepo
	logger *zap.Logger
}

func NewRevokePermissionService(authz Authorizer, repo PermissionRepo, log *zap.Logger) *RevokePermissionService {
	return &RevokePermissionService{authz: authz, repo: repo, logger: log}
}

// Revoking permissions.manage needs a super-admin, the same as granting it,
// so managers cannot remove each other.
func (s *RevokePermissionService) Revoke(ctx context.Context, actorID, id uuid.UUID) error {
	if err := requireManager(ctx, s.authz, actorID); err != nil {
		return err
	}
	p, err := s.repo.GetPermission(ctx, id)
	if err != nil {
		return fmt.Errorf("revoke permission: %w", err)
	}
	if p.Permission == common.PERMISSION_PERMISSIONS_MANAGE {
		if err := authorizeGrants(ctx, s.authz, actorID, []Grant{p.grant()}); err != nil {
			return err
		}
	}
	if err := s.repo.DeletePermission(ctx, id); err != nil {
		return fmt.Errorf("revoke permission: %w", err)
	}
	s.logger.Info("permission revoked", zap.Stringer("actor_id", actorID), zap.Stringer("permission_id", id),
		zap.Stringer("user_id", p.UserID), zap.String("permission", string(p.Permission)))
	return nil
}
