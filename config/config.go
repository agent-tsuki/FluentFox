// Package config loads, validates, and exposes all application configuration.
// Viper reads from .env.{APP_ENV} files in development/test and from real
// environment variables in production. Nothing in the app calls os.Getenv directly.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/fluentfox/api/internal/common"
)

// Config holds every typed environment variable the application needs.
type Config struct {
	// Server
	Port           string
	AppEnv         string
	AppURL         string
	AllowedOrigins []string
	TrustedProxies []string // empty: trust no proxy, use the direct peer address

	// Database
	DatabaseURL string
	DBMaxConns  int32
	DBMinConns  int32

	// JWT
	JWTAccessSecret        string
	JWTAccessExpiryMinutes int
	JWTRefreshExpiryDays   int

	// Social OAuth providers and Redis-backed, one-time hand-offs. A provider
	// is disabled unless its own credentials and the shared state store are
	// complete, so an incomplete rollout cannot affect password sign-in.
	GoogleOAuthClientID           string
	GoogleOAuthClientSecret       string
	GoogleOAuthRedirectURL        string
	XOAuthConsumerKey             string
	XOAuthConsumerSecret          string
	XOAuthCallbackURL             string
	UpstashRedisRESTURL           string
	UpstashRedisRESTToken         string
	OAuthTransactionEncryptionKey string

	// Email: Resend when ResendAPIKey is set, otherwise Gmail SMTP.
	ResendAPIKey        string
	SMTPHost            string
	SMTPPort            string
	SMTPUsername        string
	SMTPPassword        string
	EmailFromAddress    string
	EmailMXCheckEnabled bool // DNS mail-server check on or off.

	// Storage (Cloudflare R2)
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2BucketName string
	R2PublicURL  string

	// Public base URL of the kana and kanji stroke JSON files; sent to clients as-is.
	StrokesBaseURL string

	// Logging
	LogLevel string
}

type DatabaseConfig struct {
	DatabaseURL string
	DBMaxConns  int32
	DBMinConns  int32
}

// Load reads APP_ENV, loads the matching .env file for dev/test environments,
// then maps all variables into a Config struct.
// Panics if any required variable is missing — the app must not start partially configured.
func Load() *Config {
	v := viper.New()
	v.AutomaticEnv()

	appEnv := v.GetString("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	switch appEnv {
	case "development":
		v.SetConfigFile(".env.development")
		v.SetConfigType("dotenv")
		if err := v.ReadInConfig(); err != nil {
			panic(fmt.Sprintf("config: failed to load .env.development: %v", err))
		}
	case "test":
		v.SetConfigFile(".env.test")
		v.SetConfigType("dotenv")
		if err := v.ReadInConfig(); err != nil {
			panic(fmt.Sprintf("config: failed to load .env.test: %v", err))
		}
	case "production":
		// Production reads directly from the injected environment (Railway, etc.).
	default:
		panic(fmt.Sprintf("config: unknown APP_ENV value: %q", appEnv))
	}

	cfg := &Config{
		Port:   v.GetString("PORT"),
		AppEnv: appEnv,
		AppURL: require(v, "APP_URL"),

		DatabaseURL: require(v, "DATABASE_URL"),
		DBMaxConns:  int32(requireInt(v, "DB_MAX_CONNS")),
		DBMinConns:  int32(requireInt(v, "DB_MIN_CONNS")),

		JWTAccessSecret:        require(v, "JWT_ACCESS_SECRET"),
		JWTAccessExpiryMinutes: requireInt(v, "JWT_ACCESS_EXPIRY_MINUTES"),
		JWTRefreshExpiryDays:   requireInt(v, "JWT_REFRESH_EXPIRY_DAYS"),

		GoogleOAuthClientID:           v.GetString("GOOGLE_OAUTH_CLIENT_ID"),
		GoogleOAuthClientSecret:       v.GetString("GOOGLE_OAUTH_CLIENT_SECRET"),
		GoogleOAuthRedirectURL:        v.GetString("GOOGLE_OAUTH_REDIRECT_URL"),
		XOAuthConsumerKey:             v.GetString("X_OAUTH_CONSUMER_KEY"),
		XOAuthConsumerSecret:          v.GetString("X_OAUTH_CONSUMER_SECRET"),
		XOAuthCallbackURL:             v.GetString("X_OAUTH_CALLBACK_URL"),
		UpstashRedisRESTURL:           v.GetString("UPSTASH_REDIS_REST_URL"),
		UpstashRedisRESTToken:         v.GetString("UPSTASH_REDIS_REST_TOKEN"),
		OAuthTransactionEncryptionKey: v.GetString("OAUTH_TRANSACTION_ENCRYPTION_KEY"),

		SMTPHost:         require(v, "SMTP_HOST"),
		SMTPPort:         require(v, "SMTP_PORT"),
		SMTPUsername:     require(v, "SMTP_USERNAME"),
		SMTPPassword:     require(v, "SMTP_APP_PASSWORD"),
		EmailFromAddress: require(v, "EMAIL_FROM_ADDRESS"),
		ResendAPIKey:     v.GetString("RESEND_API_KEY"),

		R2AccountID:  v.GetString("R2_ACCOUNT_ID"),
		R2AccessKey:  v.GetString("R2_ACCESS_KEY"),
		R2SecretKey:  v.GetString("R2_SECRET_KEY"),
		R2BucketName: v.GetString("R2_BUCKET_NAME"),
		R2PublicURL:  v.GetString("R2_PUBLIC_URL"),

		StrokesBaseURL: v.GetString("STROKES_BASE_URL"),

		LogLevel: v.GetString("LOG_LEVEL"),
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if len(cfg.JWTAccessSecret) < common.JWT_SECRET_MIN_BYTES {
		panic(fmt.Sprintf("config: JWT_ACCESS_SECRET must be at least %d bytes", common.JWT_SECRET_MIN_BYTES))
	}

	v.SetDefault("EMAIL_MX_CHECK_ENABLED", true)
	cfg.EmailMXCheckEnabled = v.GetBool("EMAIL_MX_CHECK_ENABLED")

	if raw := v.GetString("TRUSTED_PROXIES"); raw != "" {
		cfg.TrustedProxies = strings.Split(raw, ",")
	}

	rawOrigins := require(v, "ALLOWED_ORIGINS")
	cfg.AllowedOrigins = strings.Split(rawOrigins, ",")

	sharedOAuthValues := []string{
		cfg.UpstashRedisRESTURL,
		cfg.UpstashRedisRESTToken,
		cfg.OAuthTransactionEncryptionKey,
	}
	validateOptionalGroup("OAuth state storage", sharedOAuthValues,
		"UPSTASH_REDIS_REST_URL, UPSTASH_REDIS_REST_TOKEN, and OAUTH_TRANSACTION_ENCRYPTION_KEY")
	validateOptionalGroup("Google OAuth", []string{
		cfg.GoogleOAuthClientID,
		cfg.GoogleOAuthClientSecret,
		cfg.GoogleOAuthRedirectURL,
	}, "GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET, and GOOGLE_OAUTH_REDIRECT_URL")
	validateOptionalGroup("X OAuth", []string{
		cfg.XOAuthConsumerKey,
		cfg.XOAuthConsumerSecret,
		cfg.XOAuthCallbackURL,
	}, "X_OAUTH_CONSUMER_KEY, X_OAUTH_CONSUMER_SECRET, and X_OAUTH_CALLBACK_URL")

	if (hasConfiguredValue([]string{cfg.GoogleOAuthClientID, cfg.GoogleOAuthClientSecret, cfg.GoogleOAuthRedirectURL}) ||
		hasConfiguredValue([]string{cfg.XOAuthConsumerKey, cfg.XOAuthConsumerSecret, cfg.XOAuthCallbackURL})) && !cfg.OAuthStateStoreEnabled() {
		panic("config: social OAuth requires UPSTASH_REDIS_REST_URL, UPSTASH_REDIS_REST_TOKEN, and OAUTH_TRANSACTION_ENCRYPTION_KEY")
	}

	return cfg
}

func LoadDatabase() *DatabaseConfig {
	v := viper.New()
	v.AutomaticEnv()

	return &DatabaseConfig{
		DatabaseURL: require(v, "DATABASE_URL"),
		DBMaxConns:  int32(requireInt(v, "DB_MAX_CONNS")),
		DBMinConns:  int32(requireInt(v, "DB_MIN_CONNS")),
	}
}

func (c *Config) IsDevelopment() bool { return c.AppEnv == "development" }
func (c *Config) IsProduction() bool  { return c.AppEnv == "production" }
func (c *Config) IsTest() bool        { return c.AppEnv == "test" }

// OAuthStateStoreEnabled reports whether encrypted, cross-instance OAuth state
// is available for a provider flow.
func (c *Config) OAuthStateStoreEnabled() bool {
	return c.UpstashRedisRESTURL != "" && c.UpstashRedisRESTToken != "" &&
		c.OAuthTransactionEncryptionKey != ""
}

// GoogleOAuthEnabled is deliberately all-or-nothing. It lets us deploy the
// routes before credentials are provisioned while failing closed on use.
func (c *Config) GoogleOAuthEnabled() bool {
	return c.GoogleOAuthClientID != "" && c.GoogleOAuthClientSecret != "" &&
		c.GoogleOAuthRedirectURL != "" && c.OAuthStateStoreEnabled()
}

// XOAuthEnabled uses OAuth 1.0a consumer credentials plus the shared,
// encrypted transaction store used to bind the browser callback safely.
func (c *Config) XOAuthEnabled() bool {
	return c.XOAuthConsumerKey != "" && c.XOAuthConsumerSecret != "" &&
		c.XOAuthCallbackURL != "" && c.OAuthStateStoreEnabled()
}

func validateOptionalGroup(name string, values []string, required string) {
	configured := 0
	for _, value := range values {
		if value != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(values) {
		panic(fmt.Sprintf("config: %s requires %s together", name, required))
	}
}

func hasConfiguredValue(values []string) bool {
	for _, value := range values {
		if value != "" {
			return true
		}
	}
	return false
}

func require(v *viper.Viper, key string) string {
	val := v.GetString(key)
	if val == "" {
		panic(fmt.Sprintf("config: required variable %q is not set", key))
	}
	return val
}

func requireInt(v *viper.Viper, key string) int {
	val := v.GetInt(key)
	if val == 0 && v.GetString(key) == "" {
		panic(fmt.Sprintf("config: required variable %q is not set", key))
	}
	return val
}
