package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/services"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/token"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo     UserRepo
	argon        Argon2Config
	tokenService *TokenService
	mailService  *services.MailService
	logger       *zap.Logger
}

// ResendVerificationService handles the resend-verification flow.
// It replaces the old pending token with a fresh one and fires the email async.
type ResendVerificationService struct {
	userRepo     UserRepo
	tokenService *TokenService
	mailService  *services.MailService
	logger       *zap.Logger
}

type TokenService struct {
	argon Argon2Config
}

type TokenVerificationService struct {
	userRepo UserRepo
	logger   *zap.Logger
}

type LoginService struct {
	userRepo UserRepo
	argon    Argon2Config
	maker    *token.Maker
	logger   *zap.Logger
}

func NewTokenVerificationService(userRepo UserRepo, log *zap.Logger) *TokenVerificationService {
	return &TokenVerificationService{userRepo: userRepo, logger: log}
}

func NewAuthService(userRepo UserRepo, mailService *services.MailService, log *zap.Logger) *AuthService {
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	ts := &TokenService{argon: cfg}
	return &AuthService{userRepo: userRepo, argon: cfg, tokenService: ts, mailService: mailService, logger: log}
}

func NewResendVerificationService(userRepo UserRepo, mailService *services.MailService, log *zap.Logger) *ResendVerificationService {
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	ts := &TokenService{argon: cfg}
	return &ResendVerificationService{userRepo: userRepo, tokenService: ts, mailService: mailService, logger: log}
}

func NewLogin(userRepo UserRepo, maker *token.Maker, log *zap.Logger) *LoginService {
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	return &LoginService{userRepo: userRepo, argon: cfg, maker: maker, logger: log}
}

func (s *AuthService) registerUser(ctx context.Context, req RegisterRequest) error {
	s.logger.Info("Registering new user", zap.String("email", req.Email))
	if err := s.validateUserCredential(ctx, &req); err != nil {
		return err
	}

	hashedData, err := s.tokenService.generateAuthToken()
	if err != nil {
		return err
	}

	hashPassword, err := s.argon.hashStringWithSalt(req.Password)
	if err != nil {
		return err
	}

	if err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		return s.createUser(ctx, tx, hashPassword, hashedData["hashed"], req)
	}); err != nil {
		return err
	}

	s.mailService.SendVerificationEmailAsync(req.Email, req.FirstName, hashedData["token"])
	return nil
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
		s.logger.Warn("Error while fetching user with email")
		return exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error while fetching email", err)
	}
	if exist {
		s.logger.Error("User with same email already exist email: ",zap.String("email", req.Email))
		return exceptions.ErrEmailAlreadyInUse
	}

	if req.UserName != nil && *req.UserName != ""{
		exist, err := s.userRepo.GetExistingUserForUsername(ctx, *req.UserName)
		s.logger.Info("Fetching user with username: ",zap.String("username", *req.UserName))
		if err != nil {
			return exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error while fetching username", err)
		}
		if exist {
			s.logger.Error("User with same username already exist username: ",zap.String("email", *req.UserName))
			return exceptions.ErrUsernameAlreadyInUse
		}
	} else {
		generated, err := s.getGenerateUsername(ctx, *req)
		s.logger.Info("Generated new user name username: ", zap.String("username", generated))
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


func (t *LoginService) Login(ctx context.Context, loginReq LoginRequest) (LoginResponse, error) {
	t.logger.Info("Attempting login", zap.String("email", loginReq.Email))

	// Fetch user by email
	userData, err := t.userRepo.GetUserForEmail(ctx, loginReq.Email)
	if err != nil {
		t.logger.Warn("Login failed: email not found", zap.String("email", loginReq.Email))
		return LoginResponse{}, exceptions.ErrUserEmailNotFound
	}

	// Validate user
	if err = t.ValidateUserDetail(userData); err != nil {
		t.logger.Warn("Login failed while validating user")
		return LoginResponse{}, err
	}

	// Get salt from stored hash
	salt, err := GetHashedSalt(userData.PasswordHash)
	if err != nil {
		t.logger.Error("Failed to parse stored hash", zap.Error(err))
		return LoginResponse{}, err
	}
	saltByte, err := DecodeHashSalt(salt)
	if err != nil {
		t.logger.Error("Failed to decode hashed salt")
		return LoginResponse{}, err
	}

	// Hash incoming password and compare
	incomingHash := t.argon.hashedString(loginReq.Password, saltByte)
	if subtle.ConstantTimeCompare([]byte(incomingHash), []byte(userData.PasswordHash)) != 1 {
		t.logger.Warn("Login failed: password mismatch", zap.String("email", loginReq.Email))
		return LoginResponse{}, exceptions.ErrInvalidCredentials
	}

	// Issue tokens
	resp, err := t.JetTokenAssigner(ctx, userData)
	if err != nil {
		return LoginResponse{}, err
	}

	t.logger.Info("User logged in successfully", zap.String("email", loginReq.Email))
	return resp, nil
}

func (t *LoginService) ValidateUserDetail(userData users.User) error{
	if !userData.IsEmailVerified{
		t.logger.Warn("User is not verified for email")
		return exceptions.Wrap(http.StatusUnauthorized, "UNAUTHORIZE", "Please verify email first", nil)
	}
	if !userData.IsActive{
		t.logger.Warn("User account is suspended")
		return exceptions.Wrap(http.StatusUnauthorized, "UNAUTHORIZE", "Your account is suspended contact, customer care", nil)
	}
	if userData.IsDeleted{
		t.logger.Warn("User account is deleted")
		return exceptions.Wrap(http.StatusUnauthorized, "UNAUTHORIZE", "You don't have any account any longer contact, customer care", nil)
	}
	return nil
}

func (t *LoginService) JetTokenAssigner(ctx context.Context, userData users.User) (LoginResponse, error) {
	// Generate short-lived JWT access token
	accessToken, err := t.maker.GenerateAccessToken(userData.ID, userData.IsAdmin, userData.IsEmailVerified)
	if err != nil {
		t.logger.Info("Error while generating JWT token", zap.String("Error", err.Error()))
		return LoginResponse{}, err
	}

	// Generate long-lived opaque refresh token
	refreshToken, err := t.maker.GenerateRefreshToken()
	if err != nil {
		t.logger.Info("Error while generating refresh token", zap.String("Error", err.Error()))
		return LoginResponse{}, err
	}

	hashedRefresh := hashVerificationToken(refreshToken)

	expireAt := t.maker.RefreshExpiryTime()
	if _, err := t.userRepo.UpsertRefreshToken(ctx, userData.ID, hashedRefresh, expireAt); err != nil {
		t.logger.Info("Error while storing refresh token", zap.String("Error", err.Error()))
		return LoginResponse{}, err
	}

	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserSummary{
			UserID:        userData.ID,
			Username:      userData.Username,
			IsAdmin:       userData.IsAdmin,
			EmailVerified: userData.IsEmailVerified,
			IsActive:      userData.IsActive,
		},
	}, nil
}

func (t *LoginService) RefreshToken(ctx context.Context, raw string) (LoginResponse, error) {
	if raw == "" {
		return LoginResponse{}, exceptions.ErrBadRequest
	}

	hash := hashVerificationToken(raw)
	record, err := t.userRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		t.logger.Warn("Refresh failed: token not found")
		return LoginResponse{}, exceptions.ErrTokenInvalid
	}

	if record.IsRevoked {
		t.logger.Warn("Refresh failed: token is revoked", zap.Stringer("user_id", record.UserID))
		return LoginResponse{}, exceptions.ErrTokenInvalid
	}

	if time.Now().UTC().After(record.ExpiresAt) {
		t.logger.Warn("Refresh failed: token expired", zap.Stringer("user_id", record.UserID))
		return LoginResponse{}, exceptions.ErrTokenExpired
	}

	userData, err := t.userRepo.GetUserByID(ctx, record.UserID)
	if err != nil {
		t.logger.Warn("Refresh failed: user not found", zap.Stringer("user_id", record.UserID))
		return LoginResponse{}, exceptions.ErrUserNotFound
	}

	if err = t.ValidateUserDetail(userData); err != nil {
		return LoginResponse{}, err
	}

	// Rotate: issue a brand-new token pair
	return t.JetTokenAssigner(ctx, userData)
}


func (s *ResendVerificationService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.GetUserForEmail(ctx, email)
	if err != nil {
		return exceptions.ErrUserEmailNotFound
	}

	if user.IsEmailVerified {
		return exceptions.Wrap(http.StatusConflict, "OPERATION_ALREADY_PERFORMED", "email is already verified", nil)
	}

	profile, err := s.userRepo.GetUserProfileByUserID(ctx, user.ID)
	if err != nil {
		return exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error fetching user profile", err)
	}

	hashedData, err := s.tokenService.generateAuthToken()
	if err != nil {
		return err
	}

	expireAt := s.tokenService.authTokenExpireTime()
	if err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		return s.userRepo.ReplaceVerificationToken(ctx, tx, user.ID, hashedData["hashed"], &expireAt)
	}); err != nil {
		return err
	}

	s.mailService.SendVerificationEmailAsync(user.Email, profile.FirstName, hashedData["token"])
	s.logger.Info("resend verification fired", zap.String("email", email))
	return nil
}

func (t *LoginService) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return exceptions.ErrBadRequest
	}

	hash := hashVerificationToken(raw)
	record, err := t.userRepo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		t.logger.Warn("Logout failed: token not found")
		return exceptions.ErrTokenInvalid
	}

	if err := t.userRepo.RevokeRefreshToken(ctx, record.UserID); err != nil {
		t.logger.Error("Logout failed: could not revoke token", zap.Error(err))
		return err
	}

	t.logger.Info("User logged out", zap.Stringer("user_id", record.UserID))
	return nil
}