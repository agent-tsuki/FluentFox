// Package config loads, validates, and exposes all application configuration.
// It is the single source of truth for environment variables.
// Nothing in the application reads os.Getenv directly — everything comes through here.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every typed environment variable the application needs.
// It is constructed once at startup and passed as a dependency to every
// component that needs it. There are no global Config instances.
type Config struct {
	// Server
	Port           string
	AppEnv         string
	AppURL         string
	AllowedOrigins []string

	// Database
	DatabaseURL string
	DBMaxConns  int32
	DBMinConns  int32

	// JWT
	JWTAccessSecret          string
	JWTRefreshSecret         string
	JWTAccessExpiryMinutes   int
	JWTRefreshExpiryDays     int

	// Email
	ResendAPIKey       string
	EmailFromAddress   string

	// Storage (Cloudflare R2)
	R2AccountID   string
	R2AccessKey   string
	R2SecretKey   string
	R2BucketName  string
	R2PublicURL   string

	// Logging
	LogLevel string
}

// Load reads APP_ENV, loads the corresponding .env file if needed,
// and then parses every required variable into a Config struct.
// It panics if any required variable is missing or malformed —
// the app must never start with a partial configuration.
func Load() *Config {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	switch appEnv {
	case "development":
		if err := godotenv.Load(".env.development"); err != nil {
			panic(fmt.Sprintf("config: failed to load .env.development: %v", err))
		}
	case "test":
		if err := godotenv.Load(".env.test"); err != nil {
			panic(fmt.Sprintf("config: failed to load .env.test: %v", err))
		}
	case "production":
		// Production reads directly from the real environment (Railway injects vars).
		// No file is loaded.
	default:
		panic(fmt.Sprintf("config: unknown APP_ENV value: %q", appEnv))
	}

	cfg := &Config{
		Port:   requireEnv("PORT"),
		AppEnv: appEnv,
		AppURL: requireEnv("APP_URL"),

		DatabaseURL: requireEnv("DATABASE_URL"),
		DBMaxConns:  int32(requireEnvInt("DB_MAX_CONNS")),
		DBMinConns:  int32(requireEnvInt("DB_MIN_CONNS")),

		JWTAccessSecret:        requireEnv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:       requireEnv("JWT_REFRESH_SECRET"),
		JWTAccessExpiryMinutes: requireEnvInt("JWT_ACCESS_EXPIRY_MINUTES"),
		JWTRefreshExpiryDays:   requireEnvInt("JWT_REFRESH_EXPIRY_DAYS"),

		ResendAPIKey:     requireEnv("RESEND_API_KEY"),
		EmailFromAddress: requireEnv("EMAIL_FROM_ADDRESS"),

		R2AccountID:  requireEnv("R2_ACCOUNT_ID"),
		R2AccessKey:  requireEnv("R2_ACCESS_KEY"),
		R2SecretKey:  requireEnv("R2_SECRET_KEY"),
		R2BucketName: requireEnv("R2_BUCKET_NAME"),
		R2PublicURL:  requireEnv("R2_PUBLIC_URL"),

		LogLevel: getEnvWithDefault("LOG_LEVEL", "info"),
	}

	rawOrigins := requireEnv("ALLOWED_ORIGINS")
	cfg.AllowedOrigins = strings.Split(rawOrigins, ",")

	return cfg
}

// IsDevelopment reports whether the app is running in development mode.
func (c *Config) IsDevelopment() bool { return c.AppEnv == "development" }

// IsProduction reports whether the app is running in production mode.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// IsTest reports whether the app is running in test mode.
func (c *Config) IsTest() bool { return c.AppEnv == "test" }

// requireEnv returns the value of the named environment variable or panics.
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("config: required environment variable %q is not set", key))
	}
	return v
}

// requireEnvInt parses the named variable as an integer or panics.
func requireEnvInt(key string) int {
	raw := requireEnv(key)
	v, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("config: environment variable %q must be an integer, got %q", key, raw))
	}
	return v
}

// getEnvWithDefault returns the environment variable value or a fallback default.
func getEnvWithDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
