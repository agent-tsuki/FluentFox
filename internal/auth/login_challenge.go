package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

const (
	loginChallengeReasonVPN = "vpn"
	emailHintMask           = "****"
	// Derives the code key from the shared secret, so a code HMAC can never be
	// confused with, or used to forge, anything else signed with that secret.
	loginCodeKeyLabel = "foxsensei login code v1"
)

type LoginCodeMailer interface {
	SendLoginCode(ctx context.Context, to, name, code, where string) error
}

// LoginChallengeService emails a one-time code when a password login comes
// from a VPN, and checks it.
type LoginChallengeService struct {
	userRepo UserRepo
	mailer   LoginCodeMailer
	codeKey  []byte
	logger   *zap.Logger
}

func NewLoginChallengeService(userRepo UserRepo, mailer LoginCodeMailer, secret string, log *zap.Logger) *LoginChallengeService {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(loginCodeKeyLabel))
	return &LoginChallengeService{userRepo: userRepo, mailer: mailer, codeKey: mac.Sum(nil), logger: log}
}

func (s *LoginChallengeService) Start(ctx context.Context, user *users.User, loc *users.SessionLocation) (LoginChallengeResponse, error) {
	profile, err := s.userRepo.GetUserProfileByUserID(ctx, user.ID)
	if err != nil {
		return LoginChallengeResponse{}, fmt.Errorf("start login challenge: load profile: %w", err)
	}
	code, err := generateLoginCode()
	if err != nil {
		return LoginChallengeResponse{}, fmt.Errorf("start login challenge: generate code: %w", err)
	}

	challengeID := uuid.New()
	now := time.Now().UTC()
	err = s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// The user lock serialises parallel logins so the limits cannot be raced.
		if _, err := s.userRepo.LockUserForUpdate(ctx, tx, user.ID); err != nil {
			return fmt.Errorf("start login challenge: lock user: %w", err)
		}
		windowStart := now.Add(-common.LOGIN_CODE_SEND_WINDOW)
		if err := s.userRepo.DeleteOldLoginChallenges(ctx, tx, user.ID, windowStart); err != nil {
			return fmt.Errorf("start login challenge: %w", err)
		}
		stats, err := s.userRepo.GetLoginChallengeStats(ctx, tx, user.ID, windowStart)
		if err != nil {
			return fmt.Errorf("start login challenge: %w", err)
		}
		if err := checkLoginCodeLimits(stats, now); err != nil {
			return err
		}
		challenge := &users.LoginChallenge{
			ID:        challengeID,
			UserID:    user.ID,
			CodeHash:  s.hashCode(challengeID, code),
			Location:  loc,
			ExpiresAt: now.Add(common.LOGIN_CODE_EXPIRES_AFTER),
		}
		if err := s.userRepo.CreateLoginChallenge(ctx, tx, challenge); err != nil {
			return fmt.Errorf("start login challenge: %w", err)
		}
		// Sent inside the transaction so a failed send leaves no unusable
		// challenge. The user row stays locked for up to EMAIL_SEND_TIMEOUT,
		// the same trade-off resend-verification makes.
		if err := s.mailer.SendLoginCode(ctx, user.Email, profile.FirstName, code, placeName(loc)); err != nil {
			return exceptions.WithCause(exceptions.ErrLoginCodeEmailSendFailed, err)
		}
		return nil
	})
	if err != nil {
		return LoginChallengeResponse{}, fmt.Errorf("start login challenge: %w", err)
	}

	s.logger.Info("login code sent", zap.Stringer("user_id", user.ID))
	return LoginChallengeResponse{
		ChallengeID:      challengeID,
		Reason:           loginChallengeReasonVPN,
		EmailHint:        maskEmail(user.Email),
		ExpiresInSeconds: int(common.LOGIN_CODE_EXPIRES_AFTER.Seconds()),
	}, nil
}

// Consume checks the code and returns who it was for and where they logged in
// from. A wrong guess is committed before its error is returned: returning
// the error inside the transaction would roll the attempt count back and make
// the attempt limit useless.
func (s *LoginChallengeService) Consume(ctx context.Context, challengeID uuid.UUID, code string) (uuid.UUID, users.SessionLocation, error) {
	var (
		userID uuid.UUID
		loc    users.SessionLocation
		wrong  bool
	)
	err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		challenge, err := s.userRepo.LockLoginChallenge(ctx, tx, challengeID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrLoginCodeInvalid
		}
		if err != nil {
			return fmt.Errorf("consume login challenge: %w", err)
		}

		now := time.Now().UTC()
		if !challengeUsable(&challenge, now) {
			return exceptions.ErrLoginCodeInvalid
		}
		given := s.hashCode(challengeID, strings.TrimSpace(code))
		if subtle.ConstantTimeCompare([]byte(challenge.CodeHash), []byte(given)) != 1 {
			wrong = true
			return s.userRepo.RecordLoginChallengeAttempt(ctx, tx, challenge.ID, challenge.Attempts+1)
		}
		if err := s.userRepo.ConsumeLoginChallenge(ctx, tx, challenge.ID, now); err != nil {
			return fmt.Errorf("consume login challenge: %w", err)
		}
		userID = challenge.UserID
		if challenge.Location != nil {
			loc = *challenge.Location
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, users.SessionLocation{}, fmt.Errorf("consume login challenge: %w", err)
	}
	if wrong {
		return uuid.Nil, users.SessionLocation{}, exceptions.ErrLoginCodeInvalid
	}
	return userID, loc, nil
}

func challengeUsable(c *users.LoginChallenge, now time.Time) bool {
	return c.ConsumedAt == nil && !now.After(c.ExpiresAt) && c.Attempts < common.LOGIN_CODE_MAX_ATTEMPTS
}

func checkLoginCodeLimits(stats users.LoginChallengeStats, now time.Time) error {
	if stats.Latest != nil {
		if wait := stats.Latest.Add(common.LOGIN_CODE_COOLDOWN).Sub(now); wait > 0 {
			return exceptions.WithRetryAfter(exceptions.ErrLoginCodeCooldown, wait)
		}
	}
	if stats.Count >= common.LOGIN_CODE_DAILY_LIMIT {
		return exceptions.ErrLoginCodeDailyLimit
	}
	return nil
}

// Binding the challenge id into the HMAC means a code only works for the
// challenge it was sent for.
func (s *LoginChallengeService) hashCode(challengeID uuid.UUID, code string) string {
	mac := hmac.New(sha256.New, s.codeKey)
	_, _ = mac.Write([]byte(challengeID.String() + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateLoginCode() (string, error) {
	limit := big.NewInt(int64(math.Pow10(common.LOGIN_CODE_LENGTH)))
	n, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", fmt.Errorf("random login code: %w", err)
	}
	return fmt.Sprintf("%0*d", common.LOGIN_CODE_LENGTH, n.Int64()), nil
}

// "shivam@gmail.com" becomes "s****@gmail.com", so the user can tell which
// inbox to check without the response revealing the full address.
func maskEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return ""
	}
	return string([]rune(local)[:1]) + emailHintMask + "@" + domain
}

// "Tokyo, Kanto, JP", or "" when nothing is known.
func placeName(loc *users.SessionLocation) string {
	candidates := []string{loc.City, loc.Region, loc.CountryCode}
	parts := make([]string, 0, len(candidates))
	for _, p := range candidates {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}
