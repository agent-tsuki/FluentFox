package auth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	googleProvider             = "google"
	googleAuthorizeEndpoint    = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint        = "https://oauth2.googleapis.com/token"
	googleJWKSURL              = "https://www.googleapis.com/oauth2/v3/certs"
	googleOAuthStateTTL        = 10 * time.Minute
	googleOAuthExchangeCodeTTL = time.Minute
	oauthHTTPTimeout           = 10 * time.Second
	oauthStateCookieName       = "ff_google_oauth_state"
)

var (
	errGoogleOAuthUnavailable = exceptions.New(http.StatusServiceUnavailable, "OAUTH_UNAVAILABLE", "Google sign-in is not configured")
	errGoogleOAuthFailed      = exceptions.New(http.StatusUnauthorized, "OAUTH_FAILED", "Google sign-in could not be completed")
	errGoogleAccountLink      = exceptions.New(http.StatusConflict, "OAUTH_ACCOUNT_LINK_REQUIRED", "an account already exists for this email; sign in with your password first")
)

// GoogleOAuthService owns Google authorization-code sign-in.  It deliberately
// persists only short-lived, encrypted state and one-time hand-off codes in
// Redis; Vercel functions must not keep either in process memory.
type GoogleOAuthService struct {
	clientID     string
	clientSecret string
	redirectURL  string
	frontendURL  string
	cookieSecure bool

	store    *oauthStore
	verifier *oidc.IDTokenVerifier
	http     *http.Client
	users    *users.Repository
	login    *LoginService
	logger   *zap.Logger
}

type googleOAuthTransaction struct {
	ReturnTo     string `json:"return_to"`
	CodeVerifier string `json:"code_verifier"`
	Nonce        string `json:"nonce"`
}

type oauthHandoff struct {
	Login    LoginResponse `json:"login"`
	Email    string        `json:"email"`
	ReturnTo string        `json:"return_to"`
}

type googleTokenResponse struct {
	IDToken string `json:"id_token"`
}

type googleIDTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Name          string `json:"name"`
	Locale        string `json:"locale"`
}

// OAuthExchangeRequest is sent by the frontend callback page. The code only
// lives for one minute and is consumed atomically before a session is returned.
type OAuthExchangeRequest struct {
	Code string `json:"code"`
}

// OAuthExchangeResponse has the existing login response shape plus the email
// used to populate the frontend's current user object.
type OAuthExchangeResponse struct {
	LoginResponse
	Email    string `json:"email"`
	ReturnTo string `json:"return_to"`
}

func NewGoogleOAuthService(cfg *config.Config, repo *users.Repository, login *LoginService, log *zap.Logger) (*GoogleOAuthService, error) {
	if !cfg.GoogleOAuthEnabled() {
		return nil, nil
	}

	frontendURL, err := url.Parse(cfg.AppURL)
	if err != nil || frontendURL.Scheme == "" || frontendURL.Host == "" {
		return nil, fmt.Errorf("invalid APP_URL for OAuth callback")
	}
	redirectURL, err := url.Parse(cfg.GoogleOAuthRedirectURL)
	if err != nil || redirectURL.Scheme != "https" || redirectURL.Host == "" {
		return nil, fmt.Errorf("GOOGLE_OAUTH_REDIRECT_URL must be an absolute HTTPS URL")
	}

	store, err := newOAuthStore(cfg.UpstashRedisRESTURL, cfg.UpstashRedisRESTToken, cfg.OAuthTransactionEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("configure oauth store: %w", err)
	}

	// Constructing a remote key set does not make a network request. Keys are
	// fetched and cached only when Google sends us an ID token to verify.
	keySet := oidc.NewRemoteKeySet(context.Background(), googleJWKSURL)
	verifier := oidc.NewVerifier("https://accounts.google.com", keySet, &oidc.Config{
		ClientID:             cfg.GoogleOAuthClientID,
		SupportedSigningAlgs: []string{"RS256"},
	})

	return &GoogleOAuthService{
		clientID:     cfg.GoogleOAuthClientID,
		clientSecret: cfg.GoogleOAuthClientSecret,
		redirectURL:  cfg.GoogleOAuthRedirectURL,
		frontendURL:  cfg.AppURL,
		cookieSecure: cfg.IsProduction(),
		store:        store,
		verifier:     verifier,
		http:         &http.Client{Timeout: oauthHTTPTimeout},
		users:        repo,
		login:        login,
		logger:       log,
	}, nil
}

// Start creates a state/nonce/PKCE tuple before navigating to Google. State is
// random, server stored, one-use and encrypted; return_to is constrained to an
// internal frontend route so it cannot become an open redirect.
func (s *GoogleOAuthService) Start(ctx context.Context, returnTo string) (string, error) {
	if s == nil {
		return "", errGoogleOAuthUnavailable
	}

	returnTo, err := validOAuthReturnTo(returnTo)
	if err != nil {
		return "", err
	}
	state, err := oauthRandom(32)
	if err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	nonce, err := oauthRandom(32)
	if err != nil {
		return "", fmt.Errorf("generate oauth nonce: %w", err)
	}
	verifier, err := oauthRandom(48)
	if err != nil {
		return "", fmt.Errorf("generate PKCE verifier: %w", err)
	}

	transaction := googleOAuthTransaction{
		ReturnTo:     returnTo,
		CodeVerifier: verifier,
		Nonce:        nonce,
	}
	if err := s.store.put(ctx, oauthStateKey(state), transaction, googleOAuthStateTTL); err != nil {
		return "", fmt.Errorf("save oauth state: %w", err)
	}

	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id":             {s.clientID},
		"redirect_uri":          {s.redirectURL},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	return googleAuthorizeEndpoint + "?" + query.Encode(), nil
}

// CompleteCallback exchanges the provider code, verifies the signed OIDC ID
// token and issues a one-time frontend hand-off code. It never puts FluentFox
// access or refresh tokens in a URL.
func (s *GoogleOAuthService) CompleteCallback(ctx context.Context, state, code string) (string, error) {
	if s == nil {
		return "", errGoogleOAuthUnavailable
	}
	if state == "" || code == "" {
		return "", errGoogleOAuthFailed
	}

	var transaction googleOAuthTransaction
	found, err := s.store.take(ctx, oauthStateKey(state), &transaction)
	if err != nil {
		return "", fmt.Errorf("consume oauth state: %w", err)
	}
	if !found || transaction.CodeVerifier == "" || transaction.Nonce == "" {
		return "", errGoogleOAuthFailed
	}

	idTokenRaw, err := s.exchangeGoogleCode(ctx, code, transaction.CodeVerifier)
	if err != nil {
		s.logger.Warn("Google OAuth code exchange failed", zap.Error(err))
		return "", errGoogleOAuthFailed
	}
	idToken, err := s.verifier.Verify(ctx, idTokenRaw)
	if err != nil {
		s.logger.Warn("Google OAuth ID token verification failed", zap.Error(err))
		return "", errGoogleOAuthFailed
	}
	if idToken.Nonce != transaction.Nonce || idToken.Subject == "" {
		s.logger.Warn("Google OAuth nonce or subject validation failed")
		return "", errGoogleOAuthFailed
	}

	var claims googleIDTokenClaims
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("read Google ID token claims: %w", err)
	}
	if !claims.EmailVerified || strings.TrimSpace(claims.Email) == "" {
		return "", exceptions.New(http.StatusForbidden, "OAUTH_EMAIL_NOT_VERIFIED", "Google has not verified this email address")
	}

	user, err := s.findOrCreateUser(ctx, idToken.Subject, claims)
	if err != nil {
		return "", err
	}
	if err := s.login.ValidateUserDetail(&user); err != nil {
		return "", err
	}
	login, err := s.login.JetTokenAssigner(ctx, user)
	if err != nil {
		return "", fmt.Errorf("issue OAuth session: %w", err)
	}

	handoffCode, err := oauthRandom(32)
	if err != nil {
		return "", fmt.Errorf("generate OAuth handoff code: %w", err)
	}
	handoff := oauthHandoff{Login: login, Email: user.Email, ReturnTo: transaction.ReturnTo}
	if err := s.store.put(ctx, oauthExchangeKey(handoffCode), handoff, googleOAuthExchangeCodeTTL); err != nil {
		return "", fmt.Errorf("save OAuth handoff: %w", err)
	}

	return s.frontendCallbackURL(url.Values{"code": {handoffCode}, "return_to": {transaction.ReturnTo}}), nil
}

func (s *GoogleOAuthService) Exchange(ctx context.Context, code string) (OAuthExchangeResponse, string, error) {
	if s == nil {
		return OAuthExchangeResponse{}, "", errGoogleOAuthUnavailable
	}
	if code == "" {
		return OAuthExchangeResponse{}, "", exceptions.ErrBadRequest
	}
	var handoff oauthHandoff
	found, err := s.store.take(ctx, oauthExchangeKey(code), &handoff)
	if err != nil {
		return OAuthExchangeResponse{}, "", fmt.Errorf("consume OAuth handoff: %w", err)
	}
	if !found || handoff.Login.AccessToken == "" || handoff.Login.RefreshToken == "" {
		return OAuthExchangeResponse{}, "", errGoogleOAuthFailed
	}
	return OAuthExchangeResponse{LoginResponse: handoff.Login, Email: handoff.Email, ReturnTo: handoff.ReturnTo}, handoff.ReturnTo, nil
}

func (s *GoogleOAuthService) exchangeGoogleCode(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
		"redirect_uri":  {s.redirectURL},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google token endpoint returned %d", resp.StatusCode)
	}
	var tokenResponse googleTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}
	if tokenResponse.IDToken == "" {
		return "", errors.New("Google token endpoint returned no ID token")
	}
	return tokenResponse.IDToken, nil
}

func (s *GoogleOAuthService) findOrCreateUser(ctx context.Context, subject string, claims googleIDTokenClaims) (users.User, error) {
	email := normalizeEmail(claims.Email)
	identity, err := s.users.GetOAuthIdentity(ctx, googleProvider, subject)
	if err == nil {
		user, err := s.users.GetUserByID(ctx, identity.UserID)
		if err != nil {
			return users.User{}, fmt.Errorf("load OAuth user: %w", err)
		}
		if err := s.users.TouchOAuthIdentity(ctx, googleProvider, subject, &email, true, time.Now().UTC()); err != nil {
			return users.User{}, err
		}
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, err
	}

	// Never link a Google subject to an existing account based solely on a
	// matching email. A future authenticated account-settings flow can perform
	// explicit linking; silently linking here enables account takeover paths.
	if _, err := s.users.GetUserForEmail(ctx, email); err == nil {
		return users.User{}, errGoogleAccountLink
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, fmt.Errorf("check OAuth email owner: %w", err)
	}

	username, err := s.googleUsername(ctx, claims.GivenName, claims.Name, email)
	if err != nil {
		return users.User{}, err
	}
	firstName := googleFirstName(claims.GivenName, claims.Name, email)
	lastName := optionalString(claims.FamilyName)
	nativeLang := googleNativeLanguage(claims.Locale)

	// A social-only account still has a password-hash column today. Store a
	// cryptographically random Argon2 hash so password sign-in always fails
	// safely; no user knows or can derive this value.
	disabledPassword, err := oauthRandom(32)
	if err != nil {
		return users.User{}, fmt.Errorf("generate social-only password: %w", err)
	}
	argon := (&Argon2Config{}).defaultConfig()
	disabledHash, err := argon.hashStringWithSalt(disabledPassword)
	if err != nil {
		return users.User{}, fmt.Errorf("hash social-only password: %w", err)
	}

	var created users.User
	err = s.users.Transaction(ctx, func(tx *gorm.DB) error {
		userID, err := s.users.CreateUser(ctx, tx, email, username, disabledHash, nil)
		if err != nil {
			return err
		}
		if err := s.users.CreateProfile(ctx, tx, userID, firstName, lastName, nativeLang); err != nil {
			return err
		}
		if err := s.users.CreateUsersSettings(ctx, tx, userID, nil, nil); err != nil {
			return err
		}
		if err := s.users.UpdateUserForVerification(ctx, tx, userID); err != nil {
			return err
		}
		if err := s.users.CreateOAuthIdentity(ctx, tx, userID, googleProvider, subject, &email, true, time.Now().UTC()); err != nil {
			return err
		}
		created = users.User{
			ID:              userID,
			Email:           email,
			Username:        username,
			PasswordHash:    disabledHash,
			IsActive:        true,
			IsEmailVerified: true,
		}
		return nil
	})
	if err == nil {
		return created, nil
	}

	// Concurrent callbacks for the same Google account can race before the
	// identity exists. If the other transaction won, read its identity and use
	// that account; never fall back to email based linking.
	identity, identityErr := s.users.GetOAuthIdentity(ctx, googleProvider, subject)
	if identityErr == nil {
		return s.users.GetUserByID(ctx, identity.UserID)
	}
	if _, ownerErr := s.users.GetUserForEmail(ctx, email); ownerErr == nil {
		return users.User{}, errGoogleAccountLink
	}
	return users.User{}, fmt.Errorf("create OAuth user: %w", err)
}

func (s *GoogleOAuthService) googleUsername(ctx context.Context, givenName, name, email string) (string, error) {
	base := oauthUsernameBase(givenName)
	if base == "" {
		base = oauthUsernameBase(name)
	}
	if base == "" {
		base = oauthUsernameBase(strings.Split(email, "@")[0])
	}
	if base == "" {
		base = "googleuser"
	}
	for range 8 {
		raw, err := oauthRandom(5)
		if err != nil {
			return "", err
		}
		candidate := base + strings.ToLower(raw[:8])
		exists, err := s.users.GetExistingUserForUsername(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("could not generate a Google OAuth username")
}

func (s *GoogleOAuthService) frontendCallbackURL(fragment url.Values) string {
	u, _ := url.Parse(s.frontendURL)
	u.Path = strings.TrimRight(u.Path, "/") + "/auth/callback"
	u.RawQuery = ""
	u.Fragment = fragment.Encode()
	return u.String()
}

func (s *GoogleOAuthService) frontendErrorURL(code string) string {
	return s.frontendCallbackURL(url.Values{"error": {code}})
}

func validOAuthReturnTo(raw string) (string, error) {
	if raw == "" {
		return "/dashboard", nil
	}
	if len(raw) > 2048 || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "", exceptions.ErrBadRequest
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || u.Scheme != "" {
		return "", exceptions.ErrBadRequest
	}
	return raw, nil
}

func oauthRandom(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func oauthStateKey(state string) string   { return "oauth:google:state:" + oauthKeyHash(state) }
func oauthExchangeKey(code string) string { return "oauth:google:exchange:" + oauthKeyHash(code) }

func oauthKeyHash(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func googleFirstName(givenName, name, email string) string {
	for _, value := range []string{givenName, name, strings.Split(email, "@")[0]} {
		value = strings.TrimSpace(value)
		if value != "" {
			return truncateRunes(value, 100)
		}
	}
	return "Google user"
}

func googleNativeLanguage(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if len(locale) >= 2 && unicode.IsLetter(rune(locale[0])) && unicode.IsLetter(rune(locale[1])) {
		return locale[:2]
	}
	return "en"
}

func oauthUsernameBase(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
		if b.Len() >= 18 {
			break
		}
	}
	return b.String()
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

// oauthStore is a small encrypted key-value abstraction on top of Upstash's
// REST API. The JSON-command form is documented by Upstash and avoids a Redis
// TCP connection, which is ideal for stateless Vercel functions.
type oauthStore struct {
	endpoint string
	token    string
	aead     cipher.AEAD
	client   *http.Client
}

func newOAuthStore(endpoint, token, encodedKey string) (*oauthStore, error) {
	key, err := decodeOAuthKey(encodedKey)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, errors.New("UPSTASH_REDIS_REST_URL must be an absolute HTTPS URL")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("UPSTASH_REDIS_REST_TOKEN is empty")
	}
	return &oauthStore{
		endpoint: endpoint,
		token:    token,
		aead:     aead,
		client:   &http.Client{Timeout: oauthHTTPTimeout},
	}, nil
}

func (s *oauthStore) put(ctx context.Context, key string, value any, ttl time.Duration) error {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ciphertext, err := s.seal(plaintext)
	if err != nil {
		return err
	}
	result, err := s.command(ctx, []any{"SET", key, ciphertext, "EX", int(ttl.Seconds()), "NX"})
	if err != nil {
		return err
	}
	var status string
	if err := json.Unmarshal(result, &status); err != nil || status != "OK" {
		return errors.New("OAuth state collision")
	}
	return nil
}

func (s *oauthStore) take(ctx context.Context, key string, destination any) (bool, error) {
	result, err := s.command(ctx, []any{"GETDEL", key})
	if err != nil {
		return false, err
	}
	if bytes.Equal(bytes.TrimSpace(result), []byte("null")) {
		return false, nil
	}
	var ciphertext string
	if err := json.Unmarshal(result, &ciphertext); err != nil {
		return false, err
	}
	plaintext, err := s.open(ciphertext)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(plaintext, destination); err != nil {
		return false, err
	}
	return true, nil
}

func (s *oauthStore) command(ctx context.Context, command []any) (json.RawMessage, error) {
	body, err := json.Marshal(command)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
	if err != nil {
		return nil, err
	}
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK || response.Error != "" {
		return nil, fmt.Errorf("Upstash OAuth store command failed (%d): %s", resp.StatusCode, response.Error)
	}
	return response.Result, nil
}

func (s *oauthStore) seal(plaintext []byte) (string, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := s.aead.Seal(nil, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func (s *oauthStore) open(encoded string) ([]byte, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if len(payload) < s.aead.NonceSize() {
		return nil, errors.New("invalid encrypted OAuth value")
	}
	return s.aead.Open(nil, payload[:s.aead.NonceSize()], payload[s.aead.NonceSize():], nil)
}

func decodeOAuthKey(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	key, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		key, err = base64.StdEncoding.DecodeString(encoded)
	}
	if err != nil || len(key) != 32 {
		return nil, errors.New("OAUTH_TRANSACTION_ENCRYPTION_KEY must be base64-encoded 32 bytes")
	}
	return key, nil
}
