package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/token"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo     UserRepo
	argon        Argon2Config
	tokenService *TokenService
	mailService  VerificationMailer
	checker      EmailChecker
	releaser     *StaleAccountReleaser
	logger       *zap.Logger
}

type ResendVerificationService struct {
	userRepo UserRepo
	sender   *verificationSender
	logger   *zap.Logger
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
	maker    *token.Maker
	logger   *zap.Logger
}

func NewTokenVerificationService(userRepo UserRepo, log *zap.Logger) *TokenVerificationService {
	return &TokenVerificationService{userRepo: userRepo, logger: log}
}

func NewAuthService(
	userRepo UserRepo, mailService VerificationMailer, checker EmailChecker, releaser *StaleAccountReleaser, log *zap.Logger,
) *AuthService {
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	ts := &TokenService{argon: cfg}
	return &AuthService{
		userRepo: userRepo, argon: cfg, tokenService: ts, mailService: mailService,
		checker: checker, releaser: releaser, logger: log,
	}
}

func NewResendVerificationService(userRepo UserRepo, mailService VerificationMailer, log *zap.Logger) *ResendVerificationService {
	return &ResendVerificationService{userRepo: userRepo, sender: newVerificationSender(userRepo, mailService), logger: log}
}

func NewLogin(userRepo UserRepo, maker *token.Maker, log *zap.Logger) *LoginService {
	return &LoginService{userRepo: userRepo, maker: maker, logger: log}
}

type staleHolder struct {
	id       uuid.UUID
	conflict *exceptions.AppError
}

func (s *AuthService) registerUser(ctx context.Context, req RegisterRequest) error {
	req.Email = normalizeEmail(req.Email)
	s.logger.Info("Registering new user", zap.String("email", req.Email))
	if err := mapEmailCheckErr(s.checker.Check(ctx, req.Email), s.logger); err != nil {
		return err
	}

	holders, err := s.validateUserCredential(ctx, &req)
	if err != nil {
		return err
	}

	hashedData, err := s.tokenService.generateAuthToken()
	if err != nil {
		return err
	}

	release, err := acquireArgonSlot(ctx)
	if err != nil {
		return err
	}
	hashPassword, err := s.argon.hashStringWithSalt(req.Password)
	release()
	if err != nil {
		return err
	}

	// Freeing stale holders, creating the account and sending the email commit
	// together. If the email cannot be sent nothing is saved, so the user can
	// simply sign up again instead of hitting "email already in use".
	err = s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		for _, holder := range holders {
			if err := s.releaser.Release(ctx, tx, holder.id, holder.conflict); err != nil {
				return err
			}
		}
		if err := s.createUser(ctx, tx, hashPassword, hashedData["hashed"], req); err != nil {
			return err
		}
		if err := s.mailService.SendVerificationEmail(ctx, req.Email, req.FirstName, hashedData["token"]); err != nil {
			return exceptions.WithCause(exceptions.ErrEmailSendFailed, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("register user: %w", err)
	}
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

// Unverified owners are only candidates; whether they are really stale is
// re-checked under a lock inside the transaction.
func (s *AuthService) validateUserCredential(ctx context.Context, req *RegisterRequest) ([]staleHolder, error) {
	var holders []staleHolder

	emailHolder, err := s.userRepo.GetUserForEmail(ctx, req.Email)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
	case err != nil:
		s.logger.Warn("Error while fetching user with email")
		return nil, exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error while fetching email", err)
	case emailHolder.IsEmailVerified:
		s.logger.Error("User with same email already exist email: ", zap.String("email", req.Email))
		return nil, exceptions.ErrEmailAlreadyInUse
	default:
		holders = append(holders, staleHolder{id: emailHolder.ID, conflict: exceptions.ErrEmailAlreadyInUse})
	}

	nameHolder, err := s.resolveUsername(ctx, req, emailHolder.ID)
	if err != nil {
		return nil, err
	}
	if nameHolder != nil {
		holders = append(holders, *nameHolder)
	}

	// Lock rows in a fixed order so two signups releasing the same pair of
	// accounts cannot deadlock each other.
	slices.SortFunc(holders, func(a, b staleHolder) int { return bytes.Compare(a.id[:], b.id[:]) })
	return holders, nil
}

func (s *AuthService) resolveUsername(ctx context.Context, req *RegisterRequest, emailHolderID uuid.UUID) (*staleHolder, error) {
	if req.UserName != nil && *req.UserName != "" {
		return s.findUsernameHolder(ctx, *req.UserName, emailHolderID)
	}

	generated, err := s.getGenerateUsername(ctx, *req)
	s.logger.Info("Generated new user name username: ", zap.String("username", generated))
	if err != nil {
		return nil, errors.New("error while generating username")
	}
	req.UserName = &generated
	return nil, nil
}

func (s *AuthService) findUsernameHolder(ctx context.Context, username string, emailHolderID uuid.UUID) (*staleHolder, error) {
	nameHolder, err := s.userRepo.GetUserForUsername(ctx, username)
	s.logger.Info("Fetching user with username: ", zap.String("username", username))
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, nil
	case err != nil:
		return nil, exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error while fetching username", err)
	case nameHolder.IsEmailVerified:
		s.logger.Error("User with same username already exist username: ", zap.String("username", username))
		return nil, exceptions.ErrUsernameAlreadyInUse
	case nameHolder.ID == emailHolderID:
		return nil, nil
	default:
		return &staleHolder{id: nameHolder.ID, conflict: exceptions.ErrUsernameAlreadyInUse}, nil
	}
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
		// Lock the user row first, the same order change-email and resend use,
		// so verify waits for them instead of deadlocking.
		if _, err := t.userRepo.LockUserForUpdate(ctx, tx, fetchedData.UserID); err != nil {
			return fmt.Errorf("verify email: lock user: %w", err)
		}
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
	loginReq.Email = normalizeEmail(loginReq.Email)

	userData, err := t.userRepo.GetUserForEmail(ctx, loginReq.Email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Same work and same error as a wrong password, so neither the response
		// nor its timing reveals whether the email has an account.
		if _, err := verifyPassword(ctx, loginReq.Password, dummyPasswordHash); err != nil {
			return LoginResponse{}, fmt.Errorf("login: %w", err)
		}
		t.logger.Warn("Login failed: invalid credentials")
		return LoginResponse{}, exceptions.ErrInvalidCredentials
	}
	if err != nil {
		return LoginResponse{}, fmt.Errorf("login: load user: %w", err)
	}

	ok, err := verifyPassword(ctx, loginReq.Password, userData.PasswordHash)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("login: check password for user %s: %w", userData.ID, err)
	}
	if !ok {
		t.logger.Warn("Login failed: invalid credentials", zap.Stringer("user_id", userData.ID))
		return LoginResponse{}, exceptions.ErrInvalidCredentials
	}

	// Checked after the password so account status is only shown to the owner.
	if err := t.ValidateUserDetail(&userData); err != nil {
		return LoginResponse{}, err
	}

	resp, err := t.JetTokenAssigner(ctx, userData)
	if err != nil {
		return LoginResponse{}, err
	}

	t.logger.Info("User logged in successfully", zap.Stringer("user_id", userData.ID))
	return resp, nil
}

// Unverified users may log in; AuthMiddleware keeps them out of everything
// except verification routes.
func (t *LoginService) ValidateUserDetail(userData *users.User) error {
	if userData.IsDeleted {
		t.logger.Warn("Login blocked: account deleted", zap.Stringer("user_id", userData.ID))
		return exceptions.ErrAccountDeleted
	}
	if !userData.IsActive {
		t.logger.Warn("Login blocked: account suspended", zap.Stringer("user_id", userData.ID))
		return exceptions.ErrAccountSuspended
	}
	return nil
}

func (t *LoginService) JetTokenAssigner(ctx context.Context, userData users.User) (LoginResponse, error) {
	// Generate short-lived JWT access token
	accessToken, err := t.maker.GenerateAccessToken(userData.ID, userData.IsAdmin, userData.IsEmailVerified)
	if err != nil {
		t.logger.Error("Error while generating JWT token", zap.Error(err))
		return LoginResponse{}, fmt.Errorf("issue access token: %w", err)
	}

	// Generate long-lived opaque refresh token
	refreshToken, err := t.maker.GenerateRefreshToken()
	if err != nil {
		t.logger.Error("Error while generating refresh token", zap.Error(err))
		return LoginResponse{}, fmt.Errorf("issue refresh token: %w", err)
	}

	hashedRefresh := hashVerificationToken(refreshToken)

	expireAt := t.maker.RefreshExpiryTime()
	if _, err := t.userRepo.UpsertRefreshToken(ctx, userData.ID, hashedRefresh, expireAt); err != nil {
		t.logger.Error("Error while storing refresh token", zap.Error(err))
		return LoginResponse{}, fmt.Errorf("store refresh token: %w", err)
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t.logger.Warn("Refresh failed: token not found")
		return LoginResponse{}, exceptions.ErrTokenInvalid
	}
	if err != nil {
		return LoginResponse{}, fmt.Errorf("refresh: load token: %w", err)
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t.logger.Warn("Refresh failed: user not found", zap.Stringer("user_id", record.UserID))
		return LoginResponse{}, exceptions.ErrUserNotFound
	}
	if err != nil {
		return LoginResponse{}, fmt.Errorf("refresh: load user: %w", err)
	}

	if err := t.ValidateUserDetail(&userData); err != nil {
		return LoginResponse{}, err
	}

	// Rotate: issue a brand-new token pair
	return t.JetTokenAssigner(ctx, userData)
}


// Taking the user from the token, not from an email in the body, means nobody
// can trigger mail to someone else's inbox.
func (s *ResendVerificationService) ResendForUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return exceptions.ErrUserNotFound
	}
	if user.IsDeleted || !user.IsActive {
		return exceptions.ErrAccountInactive
	}
	if user.IsEmailVerified {
		return exceptions.ErrEmailAlreadyVerified
	}

	profile, err := s.userRepo.GetUserProfileByUserID(ctx, user.ID)
	if err != nil {
		return exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "error fetching user profile", err)
	}

	if err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// The lock serialises parallel resends so the cooldown cannot be raced,
		// and re-reads the email in case it was changed or verified meanwhile.
		locked, err := s.userRepo.LockUserForUpdate(ctx, tx, user.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.ErrUserNotFound // released by a concurrent signup
		}
		if err != nil {
			return fmt.Errorf("resend verification: %w", err)
		}
		if locked.IsEmailVerified {
			return exceptions.ErrEmailAlreadyVerified
		}
		return s.sender.send(ctx, tx, locked.ID, locked.Email, profile.FirstName)
	}); err != nil {
		return err
	}

	s.logger.Info("verification email resent", zap.Stringer("user_id", user.ID))
	return nil
}

func (t *LoginService) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return exceptions.ErrBadRequest
	}

	hash := hashVerificationToken(raw)
	record, err := t.userRepo.GetRefreshTokenByHash(ctx, hash)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t.logger.Warn("Logout failed: token not found")
		return exceptions.ErrTokenInvalid
	}
	if err != nil {
		return fmt.Errorf("logout: load token: %w", err)
	}

	if err := t.userRepo.RevokeRefreshToken(ctx, record.UserID); err != nil {
		t.logger.Error("Logout failed: could not revoke token", zap.Error(err))
		return fmt.Errorf("logout: %w", err)
	}

	t.logger.Info("User logged out", zap.Stringer("user_id", record.UserID))
	return nil
}