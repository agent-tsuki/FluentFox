package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/pkg/exceptions"
)

type StaleAccountReleaser struct {
	userRepo UserRepo
	logger   *zap.Logger
}

func NewStaleAccountReleaser(userRepo UserRepo, log *zap.Logger) *StaleAccountReleaser {
	return &StaleAccountReleaser{userRepo: userRepo, logger: log}
}

// Must run inside the caller's transaction so the delete and the new signup commit together.
func (r *StaleAccountReleaser) Release(ctx context.Context, tx *gorm.DB, holderID uuid.UUID, conflict *exceptions.AppError) error {
	holder, err := r.userRepo.LockUserForUpdate(ctx, tx, holderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil // already removed by a concurrent signup
	}
	if err != nil {
		return fmt.Errorf("release stale account: %w", err)
	}
	if holder.IsEmailVerified {
		return conflict
	}

	active, err := r.userRepo.HasActiveVerification(ctx, tx, holderID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("release stale account: %w", err)
	}
	if active {
		return conflict
	}

	deleted, err := r.userRepo.DeleteUnverifiedUser(ctx, tx, holderID)
	if err != nil {
		return fmt.Errorf("release stale account: %w", err)
	}
	if !deleted {
		return conflict
	}

	r.logger.Info("released stale unverified account", zap.Stringer("user_id", holderID))
	return nil
}
