package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

type ChangeEmailService struct {
	userRepo UserRepo
	checker  EmailChecker
	releaser *StaleAccountReleaser
	sender   *verificationSender
	logger   *zap.Logger
}

func NewChangeEmailService(
	userRepo UserRepo, mail VerificationMailer, checker EmailChecker, releaser *StaleAccountReleaser, log *zap.Logger,
) *ChangeEmailService {
	return &ChangeEmailService{
		userRepo: userRepo,
		checker:  checker,
		releaser: releaser,
		sender:   newVerificationSender(userRepo, mail),
		logger:   log,
	}
}

func (s *ChangeEmailService) ChangeEmail(ctx context.Context, userID uuid.UUID, req ChangeEmailRequest) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return exceptions.ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("change email: load user: %w", err)
	}
	if err := s.checkEligible(ctx, &user, req.CurrentPassword); err != nil {
		return err
	}

	newEmail := normalizeEmail(req.NewEmail)
	if newEmail == user.Email {
		return exceptions.ErrEmailUnchanged
	}
	if err := mapEmailCheckErr(s.checker.Check(ctx, newEmail), s.logger); err != nil {
		return err
	}
	holderID, err := s.findEmailHolder(ctx, newEmail)
	if err != nil {
		return err
	}

	profile, err := s.userRepo.GetUserProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("change email: load profile: %w", err)
	}

	if err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		return s.switchEmail(ctx, tx, userID, holderID, newEmail, profile.FirstName)
	}); err != nil {
		return fmt.Errorf("change email: %w", err)
	}
	return nil
}

// The password is required so a stolen access token alone cannot redirect the account.
func (s *ChangeEmailService) checkEligible(ctx context.Context, user *users.User, currentPassword string) error {
	if user.IsDeleted || !user.IsActive {
		return exceptions.ErrAccountInactive
	}
	if user.IsEmailVerified {
		return exceptions.ErrEmailAlreadyVerified
	}
	ok, err := verifyPassword(ctx, currentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("change email: %w", err)
	}
	if !ok {
		return exceptions.ErrCurrentPasswordInvalid
	}
	return nil
}

func (s *ChangeEmailService) findEmailHolder(ctx context.Context, email string) (uuid.UUID, error) {
	holder, err := s.userRepo.GetUserForEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("change email: look up new address: %w", err)
	}
	if holder.IsEmailVerified {
		return uuid.Nil, exceptions.ErrEmailAlreadyInUse
	}
	return holder.ID, nil
}

func (s *ChangeEmailService) switchEmail(ctx context.Context, tx *gorm.DB, userID, holderID uuid.UUID, newEmail, name string) error {
	self, err := s.userRepo.LockUserForUpdate(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("change email: lock user: %w", err)
	}
	if self.IsEmailVerified {
		return exceptions.ErrEmailAlreadyVerified
	}

	if holderID != uuid.Nil {
		if err := s.releaser.Release(ctx, tx, holderID, exceptions.ErrEmailAlreadyInUse); err != nil {
			return err
		}
	}

	updated, err := s.userRepo.UpdateUnverifiedUserEmail(ctx, tx, userID, newEmail)
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}
	if !updated {
		return exceptions.ErrEmailAlreadyVerified
	}

	if err := s.sender.send(ctx, tx, userID, newEmail, name); err != nil {
		return err
	}
	s.logger.Info("unverified email changed", zap.Stringer("user_id", userID))
	return nil
}
