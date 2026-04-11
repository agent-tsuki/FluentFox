package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo     UserRepo
	argon        Argon2Config
	tokenService *TokenService
	logger   *zap.Logger
}

type TokenService struct {
	argon Argon2Config
}

type TokenVerificationService struct {
	userRepo UserRepo
	logger   *zap.Logger
}

func NewTokenVerificationService(userRepo UserRepo, log *zap.Logger) *TokenVerificationService {
	return &TokenVerificationService{userRepo: userRepo, logger: log}
}

func NewAuthService(userRepo UserRepo, log *zap.Logger) *AuthService {
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	ts := &TokenService{argon: cfg}
	return &AuthService{userRepo: userRepo, argon: cfg, tokenService: ts, logger: log}
}

func (s *AuthService) registerUser(ctx context.Context, req RegisterRequest) error {
	if err := s.validateUserCredential(ctx, &req); err != nil {
		return err
	}

	hashedData, err := s.tokenService.generateAuthToken()
	if err != nil {
		return err
	}
	s.logger.Info("verification token (use this to verify email)", zap.String("token", hashedData["token"]))

	hashPassword, err := s.argon.hashedString(req.Password)
	if err != nil {
		return err
	}

	return s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		return s.createUser(ctx, tx, hashPassword, hashedData["hashed"], req)
	})
}

func (s *AuthService) createUser(ctx context.Context, tx *gorm.DB, hashPassword, hashedToken string, req RegisterRequest) error {
	userID, err := s.userRepo.CreateUser(ctx, tx, req.Email, *req.UserName, hashPassword, req.PhoneNumber)
	if err != nil {
		return err
	}

	if err := s.userRepo.CreateProfile(ctx, tx, userID, req.FirstName, req.LastName, req.NativeLang); err != nil {
		return err
	}

	if err := s.userRepo.CreateUsersSettings(ctx, tx, userID, nil, nil); err != nil {
		return err
	}

	tokenExpireAt := s.tokenService.authTokenExpireTime()
	if err := s.userRepo.CreateUserVerification(ctx, tx, userID, hashedToken, &tokenExpireAt); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) validateUserCredential(ctx context.Context, req *RegisterRequest) error {
	exist, err := s.userRepo.GetExistingUserForEmail(ctx, req.Email)
	if err != nil {
		return exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error while fetching email", err)
	}
	if exist {
		return exceptions.ErrEmailAlreadyInUse
	}

	if req.UserName != nil {
		exist, err := s.userRepo.GetExistingUserForUsername(ctx, *req.UserName)
		if err != nil {
			return exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error while fetching username", err)
		}
		if exist {
			return exceptions.ErrUsernameAlreadyInUse
		}
	} else {
		generated, err := s.getGenerateUsername(ctx, *req)
		if err != nil {
			return errors.New("error while generating username")
		}
		req.UserName = &generated
	}

	return nil
}

func (s *AuthService) getGenerateUsername(ctx context.Context, req RegisterRequest) (string, error) {
	for i := 0; i < common.USERNAME_RETRY; i++ {
		username, err := s.generateUsername(req.FirstName, req.LastName)
		if err != nil {
			continue
		}
		exist, err := s.userRepo.GetExistingUserForUsername(ctx, username)
		if err != nil {
			continue
		}
		if !exist {
			return username, nil
		}
	}
	return "", errors.New("username not generated, try again")
}

func (s *AuthService) generateUsername(firstName string, lastName *string) (string, error) {
	suffix, err := generateString(common.USERNAME_SUFFIX_LEN)
	if err != nil {
		return "", err
	}
	if lastName != nil {
		return fmt.Sprintf("%s%s%s", firstName, *lastName, suffix), nil
	}
	return fmt.Sprintf("%s%s", firstName, suffix), nil
}

func (t *TokenService) generateAuthToken() (map[string]string, error) {
    token, err := GenerateHashedToken(common.AUTH_TOKEN_LEN)
    if err != nil {
        return nil, err
    }
    return map[string]string{
        "token":  token,
        "hashed": hashVerificationToken(token),
    }, nil
}

func (t *TokenService) authTokenExpireTime() time.Time {
	return time.Now().UTC().Add(time.Duration(common.AUTH_TOKEN_EXPIRE_HOUR) * time.Hour)
}

func (t *TokenVerificationService) VerifyUserToken(ctx context.Context, token string) error {
	if token == "" {
		return exceptions.ErrBadRequest
	}

	// Hash once here — both the fetch and update must use the same value that
	// is stored in the DB (SHA-256 of the raw token sent to the user's inbox).
	hashedToken := hashVerificationToken(token)

	fetchedData, err := t.userRepo.GetVerificationToken(ctx, hashedToken)
	if err != nil {
		return exceptions.Wrap(http.StatusNotFound, "RESOURCE_NOT_FOUND", "error fetching token from db", err)
	}

	if err := t.ValidateTokenExpiryTime(fetchedData.ExpiresAt); err != nil {
		return err
	}

	if fetchedData.VerifiedAt != nil {
		return exceptions.Wrap(http.StatusConflict, "OPERATION_ALREADY_PERFORMED", "email already verified", nil)
	}

	return t.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		return t.UpdateValidatedToken(ctx, tx, hashedToken)
	})
}

func (t *TokenVerificationService) ValidateTokenExpiryTime(expiryTime time.Time) error {
	if time.Now().UTC().After(expiryTime) {
		return exceptions.ErrStatusGone
	}
	return nil
}

func (t *TokenVerificationService) UpdateValidatedToken(ctx context.Context, tx *gorm.DB, hashedToken string) error {
	userID, err := t.userRepo.UpdateVerificationToken(ctx, tx, time.Now().UTC(), hashedToken)
	if err != nil {
		return err
	}
	t.logger.Info("verification table updated")

	if err := t.userRepo.UpdateUserForVerification(ctx, tx, userID); err != nil {
		return err
	}
	t.logger.Info("user table updated for email verification")
	return nil
}
