package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/ipintel"
	"github.com/fluentfox/api/pkg/middleware"
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
	sessions     SessionIssuer
	logger       *zap.Logger
}

type SessionIssuer interface {
	JetTokenAssigner(ctx context.Context, userData users.User) (LoginResponse, error)
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
	userRepo   UserRepo
	maker      *token.Maker
	locator    IPLocator
	challenges LoginChallenger
	logger     *zap.Logger
}

// IPLocator finds where a login comes from and whether it is a VPN. Nil turns
// location and the VPN email code off.
type IPLocator interface {
	Lookup(ctx context.Context, ip netip.Addr) (ipintel.Result, error)
}

type LoginChallenger interface {
	Start(ctx context.Context, user *users.User, loc *users.SessionLocation) (LoginChallengeResponse, error)
	Consume(ctx context.Context, challengeID uuid.UUID, code string) (uuid.UUID, users.SessionLocation, error)
}

func NewTokenVerificationService(userRepo UserRepo, log *zap.Logger) *TokenVerificationService {
	return &TokenVerificationService{userRepo: userRepo, logger: log}
}

func NewAuthService(
	userRepo UserRepo, mailService VerificationMailer, checker EmailChecker, releaser *StaleAccountReleaser,
	sessions SessionIssuer, log *zap.Logger,
) *AuthService {
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	ts := &TokenService{argon: cfg}
	return &AuthService{
		userRepo: userRepo, argon: cfg, tokenService: ts, mailService: mailService,
		checker: checker, releaser: releaser, sessions: sessions, logger: log,
	}
}

func NewResendVerificationService(userRepo UserRepo, mailService VerificationMailer, log *zap.Logger) *ResendVerificationService {
	return &ResendVerificationService{userRepo: userRepo, sender: newVerificationSender(userRepo, mailService), logger: log}
}

func NewLogin(userRepo UserRepo, maker *token.Maker, locator IPLocator, challenges LoginChallenger, log *zap.Logger) *LoginService {
	return &LoginService{userRepo: userRepo, maker: maker, locator: locator, challenges: challenges, logger: log}
}

type staleHolder struct {
	id       uuid.UUID
	conflict *exceptions.AppError
}

// Signing the user in here saves the client a second request and a second
// argon2 hash on /auth/login.
func (s *AuthService) registerUser(ctx context.Context, req *RegisterRequest) (LoginResponse, error) {
	req.Email = normalizeEmail(req.Email)
	s.logger.Info("Registering new user", zap.String("email", req.Email))
	generatedName := req.UserName == nil || *req.UserName == ""

	holders, err := s.checkEmailAndCredentials(ctx, req)
	if err != nil {
		return LoginResponse{}, err
	}

	hashedData, err := s.tokenService.generateAuthToken()
	if err != nil {
		return LoginResponse{}, err
	}

	release, err := acquireArgonSlot(ctx)
	if err != nil {
		return LoginResponse{}, err
	}
	hashPassword, err := s.argon.hashStringWithSalt(req.Password)
	release()
	if err != nil {
		return LoginResponse{}, err
	}

	user, err := s.createAccount(ctx, holders, hashPassword, hashedData["hashed"], req, generatedName)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("register user: %w", err)
	}

	s.sendVerificationEmail(ctx, user.ID, req.Email, req.FirstName, hashedData)

	resp, err := s.sessions.JetTokenAssigner(ctx, user)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("register user: %w", err)
	}
	return resp, nil
}

// The DNS check and the DB lookups are independent, so they run side by side.
// The email check result wins so a bad address is reported before a conflict.
func (s *AuthService) checkEmailAndCredentials(ctx context.Context, req *RegisterRequest) ([]staleHolder, error) {
	email := req.Email
	checkDone := make(chan error, 1)
	go func() { checkDone <- mapEmailCheckErr(s.checker.Check(ctx, email), s.logger) }()

	holders, err := s.validateUserCredential(ctx, req)
	if checkErr := <-checkDone; checkErr != nil {
		return nil, checkErr
	}
	return holders, err
}

// A generated username is not checked up front; the unique index catches the
// rare clash and the whole transaction is retried with a new name.
func (s *AuthService) createAccount(
	ctx context.Context, holders []staleHolder, hashPassword, hashedToken string, req *RegisterRequest, generatedName bool,
) (users.User, error) {
	for attempt := 1; ; attempt++ {
		userID, err := s.commitAccount(ctx, holders, hashPassword, hashedToken, req)
		if err == nil {
			return users.User{ID: userID, Username: *req.UserName, IsActive: true}, nil
		}
		if !generatedName || attempt >= common.USERNAME_RETRY || !errors.Is(err, exceptions.ErrUsernameAlreadyInUse) {
			return users.User{}, err
		}
		username, err := s.generateUsername(req.FirstName, req.LastName)
		if err != nil {
			return users.User{}, fmt.Errorf("generate username: %w", err)
		}
		req.UserName = &username
	}
}

// Freeing stale holders and creating the account commit together. The email
// is sent after commit so a slow mail server never holds the row locks.
func (s *AuthService) commitAccount(
	ctx context.Context, holders []staleHolder, hashPassword, hashedToken string, req *RegisterRequest,
) (uuid.UUID, error) {
	var userID uuid.UUID
	err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		for _, holder := range holders {
			if err := s.releaser.Release(ctx, tx, holder.id, holder.conflict); err != nil {
				return err
			}
		}
		var err error
		userID, err = s.createUser(ctx, tx, hashPassword, hashedToken, req)
		return err
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create account: %w", err)
	}
	return userID, nil
}

// The account is already saved, so a failed send does not fail the signup; the
// user can resend from the verify-email screen. The unsent token is deleted so
// the account stays releasable and the first resend is not held back by the
// cooldown for an email that never went out.
func (s *AuthService) sendVerificationEmail(ctx context.Context, userID uuid.UUID, email, name string, tokens map[string]string) {
	sendErr := s.mailService.SendVerificationEmail(ctx, email, name, tokens["token"])
	if sendErr == nil {
		return
	}
	s.logger.Error("signup verification email not sent", zap.Stringer("user_id", userID), zap.Error(sendErr))

	err := s.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		return s.userRepo.DeleteUnsentVerification(ctx, tx, userID, tokens["hashed"])
	})
	if err != nil {
		s.logger.Error("delete unsent signup verification token", zap.Stringer("user_id", userID), zap.Error(err))
	}
}

func (s *AuthService) createUser(ctx context.Context, tx *gorm.DB, hashPassword, hashedToken string, req *RegisterRequest) (uuid.UUID, error) {
	userID, err := s.userRepo.CreateUser(ctx, tx, req.Email, *req.UserName, hashPassword, req.PhoneNumber)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert user: %w", err)
	}

	if err := s.userRepo.CreateProfile(ctx, tx, userID, req.FirstName, req.LastName, req.NativeLang); err != nil {
		return uuid.Nil, fmt.Errorf("insert profile: %w", err)
	}

	if err := s.userRepo.CreateUsersSettings(ctx, tx, userID, nil, nil); err != nil {
		return uuid.Nil, fmt.Errorf("insert settings: %w", err)
	}

	tokenExpireAt := s.tokenService.authTokenExpireTime()
	if err := s.userRepo.CreateUserVerification(ctx, tx, userID, hashedToken, &tokenExpireAt); err != nil {
		return uuid.Nil, fmt.Errorf("insert verification: %w", err)
	}

	return userID, nil
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

	generated, err := s.generateUsername(req.FirstName, req.LastName)
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

	loc := t.locate(ctx)
	// Someone on a VPN may be hiding where they are, so a password alone is
	// not enough: they must also prove they can read the account's email.
	if loc.IsVPN != nil && *loc.IsVPN && t.challenges != nil {
		challenge, err := t.challenges.Start(ctx, &userData, &loc)
		if err != nil {
			return LoginResponse{}, fmt.Errorf("login: %w", err)
		}
		t.logger.Info("Login needs an email code: VPN detected", zap.Stringer("user_id", userData.ID))
		return LoginResponse{LoginChallenge: &challenge}, nil
	}

	resp, err := t.openSession(ctx, &userData, &loc)
	if err != nil {
		return LoginResponse{}, err
	}

	t.logger.Info("User logged in successfully", zap.Stringer("user_id", userData.ID))
	return resp, nil
}

// VerifyLoginCode finishes a login that Login answered with a challenge.
func (t *LoginService) VerifyLoginCode(ctx context.Context, req VerifyLoginCodeRequest) (LoginResponse, error) {
	if t.challenges == nil {
		return LoginResponse{}, exceptions.ErrLoginCodeInvalid
	}
	userID, loc, err := t.challenges.Consume(ctx, req.ChallengeID, req.Code)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("verify login code: %w", err)
	}

	userData, err := t.userRepo.GetUserByID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return LoginResponse{}, exceptions.ErrLoginCodeInvalid
	}
	if err != nil {
		return LoginResponse{}, fmt.Errorf("verify login code: load user: %w", err)
	}
	// The account may have been suspended while the code was in the inbox.
	if err := t.ValidateUserDetail(&userData); err != nil {
		return LoginResponse{}, err
	}

	resp, err := t.openSession(ctx, &userData, &loc)
	if err != nil {
		return LoginResponse{}, err
	}
	t.logger.Info("User logged in with email code", zap.Stringer("user_id", userData.ID))
	return resp, nil
}

// A failed lookup must never block a login, so it only costs the location.
func (t *LoginService) locate(ctx context.Context) users.SessionLocation {
	info := middleware.RequestInfoFromContext(ctx)
	if t.locator == nil || !info.IsPublic() {
		return users.SessionLocation{}
	}
	ctx, cancel := context.WithTimeout(ctx, common.IP_LOOKUP_TIMEOUT)
	defer cancel()

	result, err := t.locator.Lookup(ctx, info.IP)
	if err != nil {
		t.logger.Warn("IP location lookup failed", zap.Error(err))
		return users.SessionLocation{}
	}
	loc := users.SessionLocation{
		CountryCode: result.CountryCode,
		Region:      result.Region,
		City:        result.City,
		Source:      users.LocationSourceIP,
		IsVPN:       &result.IsVPN,
	}
	// 0,0 means the provider has no coordinates for this address.
	if result.Latitude != 0 || result.Longitude != 0 {
		lat, lng := users.RoundCoordinate(result.Latitude), users.RoundCoordinate(result.Longitude)
		loc.Latitude, loc.Longitude = &lat, &lng
	}
	return loc
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

// JetTokenAssigner opens a session for a login that is already proven: signup,
// Google and X. Those skip the VPN code; the provider has verified the person.
func (t *LoginService) JetTokenAssigner(ctx context.Context, userData users.User) (LoginResponse, error) {
	loc := t.locate(ctx)
	return t.openSession(ctx, &userData, &loc)
}

// Each login gets its own session row so the user's other devices stay signed
// in. Pruning, the insert and last_login_at commit together.
func (t *LoginService) openSession(ctx context.Context, userData *users.User, loc *users.SessionLocation) (LoginResponse, error) {
	refreshToken, err := t.maker.GenerateRefreshToken()
	if err != nil {
		t.logger.Error("Error while generating refresh token", zap.Error(err))
		return LoginResponse{}, fmt.Errorf("issue refresh token: %w", err)
	}

	now := time.Now().UTC()
	info := middleware.RequestInfoFromContext(ctx)
	session := users.NewSession{
		UserID:    userData.ID,
		TokenHash: hashVerificationToken(refreshToken),
		ExpiresAt: t.maker.RefreshExpiryTime(),
		UserAgent: info.UserAgent,
		IPPrefix:  info.IPPrefix(),
		Location:  *loc,
	}

	var sessionID uuid.UUID
	err = t.userRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// Serialises logins of one user, so two at once cannot both slip
		// under the session limit or deadlock on each other's prune.
		if _, err := t.userRepo.LockUserForUpdate(ctx, tx, userData.ID); err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		// Keep one slot free for the session being created.
		revokedBefore := now.Add(-common.SESSION_REVOKED_RETENTION)
		if err := t.userRepo.PruneSessions(ctx, tx, userData.ID, now, revokedBefore, common.MAX_ACTIVE_SESSIONS-1); err != nil {
			return fmt.Errorf("prune sessions: %w", err)
		}
		id, err := t.userRepo.CreateSession(ctx, tx, &session, now)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		sessionID = id
		return t.userRepo.SetLastLogin(ctx, tx, userData.ID, now)
	})
	if err != nil {
		t.logger.Error("Error while storing session", zap.Error(err))
		return LoginResponse{}, fmt.Errorf("store session: %w", err)
	}

	return t.tokenPair(userData, sessionID, refreshToken)
}

func (t *LoginService) tokenPair(userData *users.User, sessionID uuid.UUID, refreshToken string) (LoginResponse, error) {
	accessToken, err := t.maker.GenerateAccessToken(userData.ID, userData.IsAdmin, userData.IsEmailVerified, sessionID)
	if err != nil {
		t.logger.Error("Error while generating JWT token", zap.Error(err))
		return LoginResponse{}, fmt.Errorf("issue access token: %w", err)
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

	return t.rotate(ctx, &record, &userData)
}

// Rotating in place keeps the same session (device) and its sid; only the
// token changes. A lost race with the same token is treated as invalid.
func (t *LoginService) rotate(ctx context.Context, record *users.RefreshToken, userData *users.User) (LoginResponse, error) {
	refreshToken, err := t.maker.GenerateRefreshToken()
	if err != nil {
		return LoginResponse{}, fmt.Errorf("refresh: issue refresh token: %w", err)
	}

	ok, err := t.userRepo.RotateSession(ctx, record.ID, record.TokenHash, hashVerificationToken(refreshToken),
		t.maker.RefreshExpiryTime(), time.Now().UTC())
	if err != nil {
		return LoginResponse{}, fmt.Errorf("refresh: rotate session: %w", err)
	}
	if !ok {
		t.logger.Warn("Refresh failed: token already rotated or revoked", zap.Stringer("user_id", record.UserID))
		return LoginResponse{}, exceptions.ErrTokenInvalid
	}
	return t.tokenPair(userData, record.ID, refreshToken)
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

	// Only this device signs out; the user's other sessions stay active.
	if err := t.userRepo.RevokeSessionByID(ctx, record.ID); err != nil {
		t.logger.Error("Logout failed: could not revoke token", zap.Error(err))
		return fmt.Errorf("logout: %w", err)
	}

	t.logger.Info("User logged out", zap.Stringer("user_id", record.UserID))
	return nil
}
