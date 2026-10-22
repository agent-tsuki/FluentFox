package access

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

type PreviewInviteService struct {
	repo   InvitePreviewRepo
	logger *zap.Logger
}

func NewPreviewInviteService(repo InvitePreviewRepo, log *zap.Logger) *PreviewInviteService {
	return &PreviewInviteService{repo: repo, logger: log}
}

// Public, so it shows only a masked email; the link token is the secret.
func (s *PreviewInviteService) Preview(ctx context.Context, token string) (InvitePreviewResponse, error) {
	inv, err := s.repo.FindInviteByHash(ctx, hashInviteToken(token))
	if err != nil {
		return InvitePreviewResponse{}, fmt.Errorf("preview invite: %w", err)
	}
	grants := make([]InvitePreviewGrant, 0, len(inv.Grants))
	for _, g := range inv.Grants {
		name, err := s.repo.ResourceName(ctx, g.Resource())
		if err != nil {
			return InvitePreviewResponse{}, fmt.Errorf("preview invite: %w", err)
		}
		grants = append(grants, InvitePreviewGrant{
			Permission: g.Permission, ResourceType: g.ResourceType, ResourceID: g.ResourceID, ResourceName: name,
		})
	}
	return InvitePreviewResponse{
		EmailMasked: maskEmail(inv.Email), Status: inv.Status(time.Now().UTC()), Grants: grants, ExpiresAt: inv.ExpiresAt,
	}, nil
}

type AcceptInviteService struct {
	repo   AcceptRepo
	users  UserLookup
	logger *zap.Logger
}

func NewAcceptInviteService(repo AcceptRepo, users UserLookup, log *zap.Logger) *AcceptInviteService {
	return &AcceptInviteService{repo: repo, users: users, logger: log}
}

// The invite row is locked while the grants are inserted, so two accepts of
// the same link cannot both succeed, and a failed insert leaves it unused.
func (s *AcceptInviteService) Accept(ctx context.Context, userID uuid.UUID, token string) (AcceptInviteResponse, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return AcceptInviteResponse{}, fmt.Errorf("accept invite: %w", err)
	}
	now := time.Now().UTC()
	var inv StaffInvite
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		inv, err = s.repo.LockInviteByHash(ctx, tx, hashInviteToken(token))
		if err != nil {
			return fmt.Errorf("lock invite: %w", err)
		}
		if err := checkAcceptable(&inv, &user, now); err != nil {
			return err
		}
		if err := s.repo.CreatePermissions(ctx, tx, invitePermissions(&inv, userID)); err != nil {
			return fmt.Errorf("insert invite grants: %w", err)
		}
		return s.repo.MarkInviteAccepted(ctx, tx, inv.ID, userID, now)
	})
	if err != nil {
		return AcceptInviteResponse{}, fmt.Errorf("accept invite: %w", err)
	}
	s.logger.Info("staff invite accepted", zap.Stringer("user_id", userID), zap.Stringer("invite_id", inv.ID))
	return AcceptInviteResponse{Grants: toGrantResponses(inv.Grants)}, nil
}

// The email check reads the database, not the token claim, so it holds even
// for a token issued before the user changed their email.
func checkAcceptable(inv *StaffInvite, user *users.User, now time.Time) error {
	if err := inv.acceptErr(now); err != nil {
		return err
	}
	if !user.IsEmailVerified || normalizeEmail(user.Email) != inv.Email {
		return exceptions.ErrInviteEmailMismatch
	}
	if user.ID == inv.InvitedBy {
		return exceptions.ErrPermissionSelfGrant
	}
	return nil
}

// Grants are credited to the inviter, who chose them.
func invitePermissions(inv *StaffInvite, userID uuid.UUID) []UserPermission {
	perms := make([]UserPermission, 0, len(inv.Grants))
	for _, g := range inv.Grants {
		perms = append(perms, UserPermission{
			UserID: userID, Permission: g.Permission, ResourceType: g.ResourceType,
			ResourceID: g.ResourceID, GrantedBy: &inv.InvitedBy,
		})
	}
	return perms
}
