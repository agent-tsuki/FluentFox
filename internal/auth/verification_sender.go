package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

// Runs inside the caller's transaction, so a failed send rolls back the new
// token and the old link keeps working.
type verificationSender struct {
	userRepo     UserRepo
	tokenService *TokenService
	mail         VerificationMailer
}

func newVerificationSender(userRepo UserRepo, mail VerificationMailer) *verificationSender {
	argon := Argon2Config{}
	return &verificationSender{
		userRepo:     userRepo,
		tokenService: &TokenService{argon: argon.defaultConfig()},
		mail:         mail,
	}
}

func (v *verificationSender) send(ctx context.Context, tx *gorm.DB, userID uuid.UUID, to, name string) error {
	now := time.Now().UTC()
	if err := v.checkLimits(ctx, tx, userID, now); err != nil {
		return err
	}

	// Old tokens are expired, not deleted, so they still count toward the limit
	// and an older link can no longer verify once a new one is sent.
	if err := v.userRepo.ExpireActiveVerifications(ctx, tx, userID, now); err != nil {
		return fmt.Errorf("rotate verification token: %w", err)
	}
	tokens, err := v.tokenService.generateAuthToken()
	if err != nil {
		return fmt.Errorf("rotate verification token: %w", err)
	}
	expireAt := v.tokenService.authTokenExpireTime()
	if err := v.userRepo.CreateUserVerification(ctx, tx, userID, tokens["hashed"], &expireAt); err != nil {
		return fmt.Errorf("rotate verification token: %w", err)
	}

	if err := v.mail.SendVerificationEmail(ctx, to, name, tokens["token"]); err != nil {
		return exceptions.WithCause(exceptions.ErrEmailSendFailed, err)
	}
	return nil
}

// The caller holds a lock on the user row, so two parallel requests cannot both pass.
func (v *verificationSender) checkLimits(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time) error {
	stats, err := v.userRepo.GetVerificationSendStats(ctx, tx, userID, now.Add(-common.VERIFICATION_SEND_WINDOW))
	if err != nil {
		return fmt.Errorf("check verification limits: %w", err)
	}

	if stats.Latest != nil {
		if wait := stats.Latest.Add(common.VERIFICATION_RESEND_COOLDOWN).Sub(now); wait > 0 {
			return exceptions.WithRetryAfter(exceptions.ErrVerificationCooldown, wait)
		}
	}
	if stats.Count >= common.VERIFICATION_DAILY_SEND_LIMIT && stats.Oldest != nil {
		wait := stats.Oldest.Add(common.VERIFICATION_SEND_WINDOW).Sub(now)
		return exceptions.WithRetryAfter(exceptions.ErrVerificationDailyLimit, wait)
	}
	return nil
}
