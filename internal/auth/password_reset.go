package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

const (
	passwordResetCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	passwordResetTokenBytes  = 32
)

var errPasswordResetCodeSuppressed = errors.New("password reset code suppressed")

type PasswordResetService struct {
	userRepo UserRepo
	mailer   PasswordResetMailer
	argon    Argon2Config
	codeKey  []byte
	logger   *zap.Logger
}

func NewPasswordResetService(userRepo UserRepo, mailer PasswordResetMailer, codeSecret string, log *zap.Logger) *PasswordResetService {
	argon := Argon2Config{}
	return &PasswordResetService{
		userRepo: userRepo,
		mailer:   mailer,
		argon:    argon.defaultConfig(),
		codeKey:  []byte(codeSecret),
		logger:   log,
	}
}

func (s *PasswordResetService) RequestCode(ctx context.Context, req ForgotPasswordRequest) error {
	email := normalizeEmail(req.Email)
	user, err := s.userRepo.GetUserForEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("request password reset code: load user: %w", err)
	}
	if user.IsDeleted || !user.IsActive {
		return nil
	}

	profile, err := s.userRepo.GetUserProfileByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("request password reset code: load profile: %w", err)
	}

	code, err := generatePasswordResetCode()
	if err != nil {
		return fmt.Errorf("request password reset code: generate code: %w", err)
	}
	hashedCode := s.hashCode(code)

	err = s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		lockedUser, err := s.userRepo.LockUserForUpdate(ctx, tx, user.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errPasswordResetCodeSuppressed
		}
		if err != nil {
			return fmt.Errorf("request password reset code: lock user: %w", err)
		}
		if lockedUser.IsDeleted || !lockedUser.IsActive {
			return errPasswordResetCodeSuppressed
		}

		now := time.Now().UTC()
		current, err := s.userRepo.GetPasswordResetForUpdate(ctx, tx, lockedUser.ID)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			current = users.PasswordReset{}
		case err != nil:
			return fmt.Errorf("request password reset code: load reset: %w", err)
		}

		sentCount, windowStartedAt, allowed := nextPasswordResetSendState(current, now)
		if !allowed {
			return errPasswordResetCodeSuppressed
		}

		if err := s.userRepo.UpsertPasswordResetCode(
			ctx,
			tx,
			lockedUser.ID,
			hashedCode,
			now.Add(common.PASSWORD_RESET_CODE_EXPIRES_AFTER),
			now,
			windowStartedAt,
			sentCount,
		); err != nil {
			return fmt.Errorf("request password reset code: store code: %w", err)
		}

		if err := s.mailer.SendPasswordResetCode(ctx, lockedUser.Email, profile.FirstName, code); err != nil {
			return exceptions.WithCause(exceptions.ErrPasswordResetEmailSendFailed, err)
		}
		return nil
	})
	if errors.Is(err, errPasswordResetCodeSuppressed) {
		return nil
	}
	if err != nil {
		if errors.Is(err, exceptions.ErrPasswordResetEmailSendFailed) {
			s.logger.Error("password reset code email failed", zap.Stringer("user_id", user.ID), zap.Error(err))
			return nil
		}
		return err
	}

	s.logger.Info("password reset code sent", zap.Stringer("user_id", user.ID))
	return nil
}

func (s *PasswordResetService) VerifyCode(ctx context.Context, req VerifyPasswordResetCodeRequest) error {
	email := normalizeEmail(req.Email)
	user, err := s.userRepo.GetUserForEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return exceptions.ErrPasswordResetCodeInvalid
	}
	if err != nil {
		return fmt.Errorf("verify password reset code: load user: %w", err)
	}
	if user.IsDeleted || !user.IsActive {
		return exceptions.ErrPasswordResetCodeInvalid
	}

	profile, err := s.userRepo.GetUserProfileByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("verify password reset code: load profile: %w", err)
	}

	resetToken, err := generatePasswordResetToken()
	if err != nil {
		return fmt.Errorf("verify password reset code: generate link token: %w", err)
	}
	hashedCode := s.hashCode(strings.ToUpper(strings.TrimSpace(req.Code)))
	hashedToken := hashVerificationToken(resetToken)

	err = s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		lockedUser, err := s.userRepo.LockUserForUpdate(ctx, tx, user.ID)
		if err != nil {
			return exceptions.ErrPasswordResetCodeInvalid
		}
		if lockedUser.IsDeleted || !lockedUser.IsActive {
			return exceptions.ErrPasswordResetCodeInvalid
		}

		reset, err := s.userRepo.GetPasswordResetForUpdate(ctx, tx, lockedUser.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrPasswordResetCodeInvalid
		}
		if err != nil {
			return fmt.Errorf("verify password reset code: load reset: %w", err)
		}

		now := time.Now().UTC()
		if reset.CodeVerifiedAt != nil || now.After(reset.CodeExpiresAt) || reset.CodeAttempts >= common.PASSWORD_RESET_MAX_CODE_ATTEMPTS {
			return exceptions.ErrPasswordResetCodeInvalid
		}
		if subtle.ConstantTimeCompare([]byte(reset.CodeHash), []byte(hashedCode)) != 1 {
			if err := s.userRepo.RecordPasswordResetCodeAttempt(ctx, tx, lockedUser.ID, reset.CodeAttempts+1, now); err != nil {
				return fmt.Errorf("verify password reset code: record failed attempt: %w", err)
			}
			return exceptions.ErrPasswordResetCodeInvalid
		}

		expiresAt := now.Add(common.PASSWORD_RESET_LINK_EXPIRES_AFTER)
		if err := s.userRepo.SetPasswordResetLink(ctx, tx, lockedUser.ID, hashedToken, expiresAt, now); err != nil {
			return fmt.Errorf("verify password reset code: store link token: %w", err)
		}
		if err := s.mailer.SendPasswordResetLink(ctx, lockedUser.Email, profile.FirstName, resetToken); err != nil {
			return exceptions.WithCause(exceptions.ErrPasswordResetEmailSendFailed, err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.logger.Info("password reset link sent", zap.Stringer("user_id", user.ID))
	return nil
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	hashedToken := hashVerificationToken(req.Token)
	userID, err := s.userRepo.FindPasswordResetUserByToken(ctx, hashedToken)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return exceptions.ErrPasswordResetTokenInvalid
	}
	if err != nil {
		return fmt.Errorf("reset password: find token: %w", err)
	}

	if err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		user, err := s.userRepo.LockUserForUpdate(ctx, tx, userID)
		if err != nil || user.IsDeleted || !user.IsActive {
			return exceptions.ErrPasswordResetTokenInvalid
		}

		reset, err := s.userRepo.GetPasswordResetForUpdate(ctx, tx, userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrPasswordResetTokenInvalid
		}
		if err != nil {
			return fmt.Errorf("reset password: load reset: %w", err)
		}

		now := time.Now().UTC()
		if reset.ResetTokenHash == nil || reset.ResetTokenExpiresAt == nil || reset.ResetTokenUsedAt != nil || now.After(*reset.ResetTokenExpiresAt) || subtle.ConstantTimeCompare([]byte(*reset.ResetTokenHash), []byte(hashedToken)) != 1 {
			return exceptions.ErrPasswordResetTokenInvalid
		}

		release, err := acquireArgonSlot(ctx)
		if err != nil {
			return fmt.Errorf("reset password: wait for password hashing: %w", err)
		}
		passwordHash, err := s.argon.hashStringWithSalt(req.NewPassword)
		release()
		if err != nil {
			return fmt.Errorf("reset password: hash password: %w", err)
		}

		consumed, err := s.userRepo.ConsumePasswordReset(ctx, tx, userID, hashedToken, passwordHash, now)
		if err != nil {
			return fmt.Errorf("reset password: consume token: %w", err)
		}
		if !consumed {
			return exceptions.ErrPasswordResetTokenInvalid
		}
		return nil
	}); err != nil {
		return err
	}

	s.logger.Info("password reset completed", zap.Stringer("user_id", userID))
	return nil
}

func nextPasswordResetSendState(reset users.PasswordReset, now time.Time) (int64, time.Time, bool) {
	if !reset.CodeLastSentAt.IsZero() && now.Before(reset.CodeLastSentAt.Add(common.PASSWORD_RESET_CODE_COOLDOWN)) {
		return 0, time.Time{}, false
	}
	if reset.CodeWindowStarted.IsZero() || !now.Before(reset.CodeWindowStarted.Add(common.PASSWORD_RESET_SEND_WINDOW)) {
		return 1, now, true
	}
	if reset.CodeSentCount >= common.PASSWORD_RESET_DAILY_SEND_LIMIT {
		return 0, time.Time{}, false
	}
	return reset.CodeSentCount + 1, reset.CodeWindowStarted, true
}

func generatePasswordResetCode() (string, error) {
	code := make([]byte, common.PASSWORD_RESET_CODE_LENGTH)
	for i := range code {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordResetCodeCharset))))
		if err != nil {
			return "", err
		}
		code[i] = passwordResetCodeCharset[index.Int64()]
	}
	return string(code), nil
}

func generatePasswordResetToken() (string, error) {
	bytes := make([]byte, passwordResetTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (s *PasswordResetService) hashCode(code string) string {
	mac := hmac.New(sha256.New, s.codeKey)
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}
