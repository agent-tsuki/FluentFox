package access

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type CreateInviteService struct {
	authz  Authorizer
	repo   InviteRepo
	mailer InviteMailer
	logger *zap.Logger
}

func NewCreateInviteService(authz Authorizer, repo InviteRepo, mailer InviteMailer, log *zap.Logger) *CreateInviteService {
	return &CreateInviteService{authz: authz, repo: repo, mailer: mailer, logger: log}
}

// The email is sent inside the transaction so a failed send leaves no invite
// behind that nobody received.
func (s *CreateInviteService) Create(ctx context.Context, actorID uuid.UUID, req *CreateInviteRequest) (InviteResponse, error) {
	grants := make(Grants, 0, len(req.Grants))
	for _, g := range req.Grants {
		grants = append(grants, g.toGrant())
	}
	if err := authorizeGrants(ctx, s.authz, actorID, grants); err != nil {
		return InviteResponse{}, err
	}
	token, hash, err := newInviteToken()
	if err != nil {
		return InviteResponse{}, err
	}
	now := time.Now().UTC()
	inv := StaffInvite{
		Email: normalizeEmail(req.Email), TokenHash: hash, InvitedBy: actorID, Grants: grants,
		ExpiresAt: now.Add(common.STAFF_INVITE_EXPIRES_AFTER), CreatedAt: now,
	}
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.CreateInvite(ctx, tx, &inv); err != nil {
			return fmt.Errorf("insert invite: %w", err)
		}
		return sendInvite(ctx, s.mailer, inv.Email, token)
	})
	if err != nil {
		return InviteResponse{}, fmt.Errorf("create invite: %w", err)
	}
	s.logger.Info("staff invite created", zap.Stringer("actor_id", actorID), zap.Stringer("invite_id", inv.ID))
	return toInviteResponse(&inv, now), nil
}

type ListInvitesService struct {
	authz  Authorizer
	repo   InviteRepo
	logger *zap.Logger
}

func NewListInvitesService(authz Authorizer, repo InviteRepo, log *zap.Logger) *ListInvitesService {
	return &ListInvitesService{authz: authz, repo: repo, logger: log}
}

func (s *ListInvitesService) List(ctx context.Context, actorID uuid.UUID) ([]InviteResponse, error) {
	if err := requireManager(ctx, s.authz, actorID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListInvites(ctx)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	now := time.Now().UTC()
	out := make([]InviteResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toInviteResponse(&rows[i], now))
	}
	return out, nil
}

type ResendInviteService struct {
	authz  Authorizer
	repo   InviteRepo
	mailer InviteMailer
	logger *zap.Logger
}

func NewResendInviteService(authz Authorizer, repo InviteRepo, mailer InviteMailer, log *zap.Logger) *ResendInviteService {
	return &ResendInviteService{authz: authz, repo: repo, mailer: mailer, logger: log}
}

// A new token replaces the old one, so the earlier email's link stops
// working. An expired invite can be resent; a revoked or accepted one cannot.
func (s *ResendInviteService) Resend(ctx context.Context, actorID, id uuid.UUID) (InviteResponse, error) {
	if err := requireManager(ctx, s.authz, actorID); err != nil {
		return InviteResponse{}, err
	}
	token, hash, err := newInviteToken()
	if err != nil {
		return InviteResponse{}, err
	}
	now := time.Now().UTC()
	var inv StaffInvite
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		inv, err = s.repo.LockInvite(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("lock invite: %w", err)
		}
		if status := inv.Status(now); status == InviteStatusRevoked || status == InviteStatusAccepted {
			return inv.acceptErr(now)
		}
		inv.TokenHash, inv.ExpiresAt = hash, now.Add(common.STAFF_INVITE_EXPIRES_AFTER)
		if err := s.repo.ReplaceInviteToken(ctx, tx, id, inv.TokenHash, inv.ExpiresAt); err != nil {
			return fmt.Errorf("replace invite token: %w", err)
		}
		return sendInvite(ctx, s.mailer, inv.Email, token)
	})
	if err != nil {
		return InviteResponse{}, fmt.Errorf("resend invite: %w", err)
	}
	s.logger.Info("staff invite resent", zap.Stringer("actor_id", actorID), zap.Stringer("invite_id", id))
	return toInviteResponse(&inv, now), nil
}

type RevokeInviteService struct {
	authz  Authorizer
	repo   InviteRepo
	logger *zap.Logger
}

func NewRevokeInviteService(authz Authorizer, repo InviteRepo, log *zap.Logger) *RevokeInviteService {
	return &RevokeInviteService{authz: authz, repo: repo, logger: log}
}

// Revoking twice is fine. An accepted invite cannot be revoked: its grants
// already exist and are removed one by one instead.
func (s *RevokeInviteService) Revoke(ctx context.Context, actorID, id uuid.UUID) error {
	if err := requireManager(ctx, s.authz, actorID); err != nil {
		return err
	}
	now := time.Now().UTC()
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		inv, err := s.repo.LockInvite(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("lock invite: %w", err)
		}
		switch inv.Status(now) {
		case InviteStatusRevoked:
			return nil
		case InviteStatusAccepted:
			return exceptions.ErrInviteUsed
		default:
			return s.repo.RevokeInvite(ctx, tx, id, now)
		}
	})
	if err != nil {
		return fmt.Errorf("revoke invite: %w", err)
	}
	s.logger.Info("staff invite revoked", zap.Stringer("actor_id", actorID), zap.Stringer("invite_id", id))
	return nil
}

func sendInvite(ctx context.Context, mailer InviteMailer, to, token string) error {
	if err := mailer.SendStaffInvite(ctx, to, token); err != nil {
		return exceptions.WithCause(exceptions.ErrInviteEmailSendFailed, err)
	}
	return nil
}
