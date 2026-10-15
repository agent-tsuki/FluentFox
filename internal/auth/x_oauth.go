package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	xProvider             = "x"
	xOAuthStateTTL        = 10 * time.Minute
	xOAuthExchangeCodeTTL = time.Minute
	xOAuthTokenCookieName = "ff_x_oauth_token"
	xRequestTokenEndpoint = "https://api.x.com/oauth/request_token"
	xAuthorizeEndpoint    = "https://api.x.com/oauth/authenticate"
	xAccessTokenEndpoint  = "https://api.x.com/oauth/access_token"
	xVerifyCredentialsURL = "https://api.x.com/1.1/account/verify_credentials.json?skip_status=true"
)

var (
	errXOAuthUnavailable = exceptions.New(http.StatusServiceUnavailable, "OAUTH_X_UNAVAILABLE", "X sign-in is not configured")
	errXOAuthFailed      = exceptions.New(http.StatusUnauthorized, "OAUTH_X_FAILED", "X sign-in could not be completed")
	errXAccountLink      = exceptions.New(http.StatusConflict, "OAUTH_X_ACCOUNT_LINK_REQUIRED", "sign in with your email, then connect X from Security settings")
	errXIdentityLinked   = exceptions.New(http.StatusConflict, "OAUTH_X_IDENTITY_ALREADY_LINKED", "this X account is already linked to another FoxSensei account")
)

// XOAuthService implements the OAuth 1.0a Sign in with X flow. Unlike Google
// OIDC, X does not reliably provide a verified email, so it only signs into an
// account that has explicitly linked its immutable X user ID.
type XOAuthService struct {
	config       *oauth1.Config
	frontendURL  string
	cookieSecure bool

	store  *oauthStore
	http   *http.Client
	users  *users.Repository
	login  *LoginService
	logger *zap.Logger
}

type xOAuthTransaction struct {
	ReturnTo      string     `json:"return_to"`
	RequestToken  string     `json:"request_token"`
	RequestSecret string     `json:"request_secret"`
	LinkUserID    *uuid.UUID `json:"link_user_id,omitempty"`
}

type xOAuthStart struct {
	AuthorizationURL string
	RequestToken     string
}

type xOAuthCallbackResult struct {
	Destination string
	IsLink      bool
}

type XOAuthLinkStatus struct {
	Connected bool `json:"connected"`
}

type xVerifyCredentialsResponse struct {
	IDStr      string `json:"id_str"`
	ScreenName string `json:"screen_name"`
}

func NewXOAuthService(cfg *config.Config, repo *users.Repository, login *LoginService, log *zap.Logger) (*XOAuthService, error) {
	frontendURL, err := url.Parse(cfg.AppURL)
	if err != nil || frontendURL.Scheme == "" || frontendURL.Host == "" {
		return nil, fmt.Errorf("invalid APP_URL for X OAuth callback")
	}
	service := &XOAuthService{
		frontendURL:  cfg.AppURL,
		cookieSecure: cfg.IsProduction(),
		users:        repo,
		login:        login,
		logger:       log,
	}
	// Keep a minimal service when credentials are absent so a browser start
	// request can return a safe frontend error page instead of raw API JSON.
	if !cfg.XOAuthEnabled() {
		return service, nil
	}
	callbackURL, err := url.Parse(cfg.XOAuthCallbackURL)
	if err != nil || callbackURL.Scheme != "https" || callbackURL.Host == "" {
		return nil, fmt.Errorf("X_OAUTH_CALLBACK_URL must be an absolute HTTPS URL")
	}

	store, err := newOAuthStore(cfg.UpstashRedisRESTURL, cfg.UpstashRedisRESTToken, cfg.OAuthTransactionEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("configure X OAuth store: %w", err)
	}
	httpClient := &http.Client{Timeout: oauthHTTPTimeout}
	service.config = &oauth1.Config{
		ConsumerKey:    cfg.XOAuthConsumerKey,
		ConsumerSecret: cfg.XOAuthConsumerSecret,
		CallbackURL:    cfg.XOAuthCallbackURL,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: xRequestTokenEndpoint,
			AuthorizeURL:    xAuthorizeEndpoint,
			AccessTokenURL:  xAccessTokenEndpoint,
		},
		HTTPClient: httpClient,
	}
	service.store = store
	service.http = httpClient
	return service, nil
}

func (s *XOAuthService) Start(ctx context.Context, returnTo string) (xOAuthStart, error) {
	return s.start(ctx, returnTo, nil)
}

func (s *XOAuthService) StartLink(ctx context.Context, userID uuid.UUID) (xOAuthStart, error) {
	if s == nil || s.config == nil {
		return xOAuthStart{}, errXOAuthUnavailable
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return xOAuthStart{}, exceptions.ErrUserNotFound
	}
	if err := s.login.ValidateUserDetail(&user); err != nil {
		return xOAuthStart{}, err
	}
	return s.start(ctx, "/profile?tab=settings", &userID)
}

func (s *XOAuthService) start(ctx context.Context, returnTo string, linkUserID *uuid.UUID) (xOAuthStart, error) {
	if s == nil || s.config == nil || s.store == nil {
		return xOAuthStart{}, errXOAuthUnavailable
	}
	returnTo, err := validOAuthReturnTo(returnTo)
	if err != nil {
		return xOAuthStart{}, err
	}

	requestToken, requestSecret, err := s.config.RequestToken()
	if err != nil {
		s.logger.Warn("X OAuth request token failed", zap.Error(err))
		return xOAuthStart{}, errXOAuthFailed
	}
	if requestToken == "" || requestSecret == "" {
		return xOAuthStart{}, errXOAuthFailed
	}
	authorizationURL, err := s.config.AuthorizationURL(requestToken)
	if err != nil {
		return xOAuthStart{}, fmt.Errorf("build X authorization URL: %w", err)
	}
	transaction := xOAuthTransaction{
		ReturnTo:      returnTo,
		RequestToken:  requestToken,
		RequestSecret: requestSecret,
		LinkUserID:    linkUserID,
	}
	if err := s.store.put(ctx, xOAuthStateKey(requestToken), transaction, xOAuthStateTTL); err != nil {
		return xOAuthStart{}, fmt.Errorf("save X oauth state: %w", err)
	}
	return xOAuthStart{AuthorizationURL: authorizationURL.String(), RequestToken: requestToken}, nil
}

func (s *XOAuthService) CompleteCallback(ctx context.Context, requestToken, verifier string) (xOAuthCallbackResult, error) {
	if s == nil || s.config == nil || s.store == nil {
		return xOAuthCallbackResult{}, errXOAuthUnavailable
	}
	if requestToken == "" || verifier == "" {
		return xOAuthCallbackResult{}, errXOAuthFailed
	}

	var transaction xOAuthTransaction
	found, err := s.store.take(ctx, xOAuthStateKey(requestToken), &transaction)
	if err != nil {
		return xOAuthCallbackResult{}, fmt.Errorf("consume X oauth state: %w", err)
	}
	if !found || transaction.RequestToken == "" || transaction.RequestSecret == "" ||
		subtle.ConstantTimeCompare([]byte(transaction.RequestToken), []byte(requestToken)) != 1 {
		return xOAuthCallbackResult{}, errXOAuthFailed
	}
	result := xOAuthCallbackResult{IsLink: transaction.LinkUserID != nil}

	accessToken, accessSecret, err := s.config.AccessToken(requestToken, transaction.RequestSecret, verifier)
	if err != nil {
		s.logger.Warn("X OAuth access token exchange failed", zap.Error(err))
		return result, errXOAuthFailed
	}
	identity, err := s.verifyCredentials(ctx, accessToken, accessSecret)
	if err != nil {
		s.logger.Warn("X OAuth identity verification failed", zap.Error(err))
		return result, errXOAuthFailed
	}

	if transaction.LinkUserID != nil {
		if err := s.linkXIdentity(ctx, *transaction.LinkUserID, identity); err != nil {
			return result, err
		}
		result.Destination = s.frontendXLinkURL("")
		return result, nil
	}

	user, err := s.findLinkedUser(ctx, identity)
	if err != nil {
		return result, err
	}
	if err := s.login.ValidateUserDetail(&user); err != nil {
		return result, err
	}
	login, err := s.login.JetTokenAssigner(ctx, user)
	if err != nil {
		return result, fmt.Errorf("issue X OAuth session: %w", err)
	}
	handoffCode, err := oauthRandom(32)
	if err != nil {
		return result, fmt.Errorf("generate X OAuth handoff code: %w", err)
	}
	handoff := oauthHandoff{Login: login, Email: user.Email, ReturnTo: transaction.ReturnTo}
	if err := s.store.put(ctx, xOAuthExchangeKey(handoffCode), handoff, xOAuthExchangeCodeTTL); err != nil {
		return result, fmt.Errorf("save X OAuth handoff: %w", err)
	}
	result.Destination = s.frontendCallbackURL(url.Values{
		"code":      {handoffCode},
		"return_to": {transaction.ReturnTo},
		"provider":  {xProvider},
	})
	return result, nil
}

func (s *XOAuthService) CancelCallback(ctx context.Context, requestToken string) (bool, error) {
	if s == nil || s.store == nil || requestToken == "" {
		return false, errXOAuthFailed
	}
	var transaction xOAuthTransaction
	found, err := s.store.take(ctx, xOAuthStateKey(requestToken), &transaction)
	if err != nil {
		return false, fmt.Errorf("consume cancelled X oauth state: %w", err)
	}
	if !found {
		return false, errXOAuthFailed
	}
	return transaction.LinkUserID != nil, nil
}

func (s *XOAuthService) Exchange(ctx context.Context, code string) (OAuthExchangeResponse, string, error) {
	if s == nil || s.store == nil {
		return OAuthExchangeResponse{}, "", errXOAuthUnavailable
	}
	if code == "" {
		return OAuthExchangeResponse{}, "", exceptions.ErrBadRequest
	}
	var handoff oauthHandoff
	found, err := s.store.take(ctx, xOAuthExchangeKey(code), &handoff)
	if err != nil {
		return OAuthExchangeResponse{}, "", fmt.Errorf("consume X OAuth handoff: %w", err)
	}
	if !found || handoff.Login.AccessToken == "" || handoff.Login.RefreshToken == "" {
		return OAuthExchangeResponse{}, "", errXOAuthFailed
	}
	return OAuthExchangeResponse{LoginResponse: handoff.Login, Email: handoff.Email, ReturnTo: handoff.ReturnTo}, handoff.ReturnTo, nil
}

func (s *XOAuthService) LinkStatus(ctx context.Context, userID uuid.UUID) (XOAuthLinkStatus, error) {
	if s == nil || s.config == nil {
		return XOAuthLinkStatus{}, errXOAuthUnavailable
	}
	_, err := s.users.GetOAuthIdentityForUser(ctx, userID, xProvider)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return XOAuthLinkStatus{}, nil
	}
	if err != nil {
		return XOAuthLinkStatus{}, err
	}
	return XOAuthLinkStatus{Connected: true}, nil
}

func (s *XOAuthService) verifyCredentials(ctx context.Context, accessToken, accessSecret string) (xVerifyCredentialsResponse, error) {
	client := s.config.Client(ctx, oauth1.NewToken(accessToken, accessSecret))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, xVerifyCredentialsURL, nil)
	if err != nil {
		return xVerifyCredentialsResponse{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return xVerifyCredentialsResponse{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return xVerifyCredentialsResponse{}, err
	}
	if response.StatusCode != http.StatusOK {
		return xVerifyCredentialsResponse{}, fmt.Errorf("X verify credentials returned %d", response.StatusCode)
	}
	var identity xVerifyCredentialsResponse
	if err := json.Unmarshal(body, &identity); err != nil {
		return xVerifyCredentialsResponse{}, err
	}
	if strings.TrimSpace(identity.IDStr) == "" {
		return xVerifyCredentialsResponse{}, errors.New("X verify credentials returned no id_str")
	}
	return identity, nil
}

func (s *XOAuthService) findLinkedUser(ctx context.Context, identity xVerifyCredentialsResponse) (users.User, error) {
	connected, err := s.users.GetOAuthIdentity(ctx, xProvider, identity.IDStr)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, errXAccountLink
	}
	if err != nil {
		return users.User{}, err
	}
	user, err := s.users.GetUserByID(ctx, connected.UserID)
	if err != nil {
		return users.User{}, fmt.Errorf("load X OAuth user: %w", err)
	}
	if err := s.users.TouchOAuthIdentity(ctx, xProvider, identity.IDStr, nil, false, time.Now().UTC()); err != nil {
		return users.User{}, err
	}
	return user, nil
}

func (s *XOAuthService) linkXIdentity(ctx context.Context, userID uuid.UUID, identity xVerifyCredentialsResponse) error {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return exceptions.ErrUserNotFound
	}
	if err := s.login.ValidateUserDetail(&user); err != nil {
		return err
	}

	connected, err := s.users.GetOAuthIdentity(ctx, xProvider, identity.IDStr)
	if err == nil {
		if connected.UserID != userID {
			return errXIdentityLinked
		}
		return s.users.TouchOAuthIdentity(ctx, xProvider, identity.IDStr, nil, false, time.Now().UTC())
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	err = s.users.Transaction(ctx, func(tx *gorm.DB) error {
		return s.users.CreateOAuthIdentity(ctx, tx, userID, xProvider, identity.IDStr, nil, false, time.Now().UTC())
	})
	if err == nil {
		return nil
	}
	connected, lookupErr := s.users.GetOAuthIdentity(ctx, xProvider, identity.IDStr)
	if lookupErr == nil {
		if connected.UserID == userID {
			return nil
		}
		return errXIdentityLinked
	}
	return fmt.Errorf("create X OAuth link: %w", err)
}

func (s *XOAuthService) frontendErrorURL(code string) string {
	return s.frontendCallbackURL(url.Values{"error": {code}, "provider": {xProvider}})
}

func (s *XOAuthService) frontendCallbackURL(fragment url.Values) string {
	u, _ := url.Parse(s.frontendURL)
	u.Path = strings.TrimRight(u.Path, "/") + "/auth/callback"
	u.RawQuery = ""
	u.Fragment = fragment.Encode()
	return u.String()
}

func (s *XOAuthService) frontendXLinkURL(errorCode string) string {
	u, _ := url.Parse(s.frontendURL)
	u.Path = strings.TrimRight(u.Path, "/") + "/profile"
	query := url.Values{"tab": {"settings"}}
	if errorCode == "" {
		query.Set("x", "linked")
	} else {
		query.Set("x_error", errorCode)
	}
	u.RawQuery = query.Encode()
	u.Fragment = ""
	return u.String()
}

func xOAuthStateKey(requestToken string) string { return "oauth:x:state:" + oauthKeyHash(requestToken) }
func xOAuthExchangeKey(code string) string      { return "oauth:x:exchange:" + oauthKeyHash(code) }
