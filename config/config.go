// Package config loads, validates, and exposes all application configuration.
// Viper reads from .env.{APP_ENV} files in development/test and from real
// environment variables in production. Nothing in the app calls os.Getenv directly.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds every typed environment variable the application needs.
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
	JWTAccessSecret        string
	JWTRefreshSecret       string
	JWTAccessExpiryMinutes int
	JWTRefreshExpiryDays   int

	// Email (Gmail SMTP)
	SMTPHost         string
	SMTPPort         string
	SMTPUsername     string
	SMTPPassword     string
	EmailFromAddress string

	// Storage (Cloudflare R2)
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2BucketName string
	R2PublicURL  string

	// Logging
	LogLevel string
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
		Port:   require(v, "PORT"),
		AppEnv: appEnv,
		AppURL: require(v, "APP_URL"),

		DatabaseURL: require(v, "DATABASE_URL"),
		DBMaxConns:  int32(requireInt(v, "DB_MAX_CONNS")),
		DBMinConns:  int32(requireInt(v, "DB_MIN_CONNS")),

		JWTAccessSecret:        require(v, "JWT_ACCESS_SECRET"),
		JWTRefreshSecret:       require(v, "JWT_REFRESH_SECRET"),
		JWTAccessExpiryMinutes: requireInt(v, "JWT_ACCESS_EXPIRY_MINUTES"),
		JWTRefreshExpiryDays:   requireInt(v, "JWT_REFRESH_EXPIRY_DAYS"),

		SMTPHost:         require(v, "SMTP_HOST"),
		SMTPPort:         require(v, "SMTP_PORT"),
		SMTPUsername:     require(v, "SMTP_USERNAME"),
		SMTPPassword:     require(v, "SMTP_APP_PASSWORD"),
		EmailFromAddress: require(v, "EMAIL_FROM_ADDRESS"),

		R2AccountID:  require(v, "R2_ACCOUNT_ID"),
		R2AccessKey:  require(v, "R2_ACCESS_KEY"),
		R2SecretKey:  require(v, "R2_SECRET_KEY"),
		R2BucketName: require(v, "R2_BUCKET_NAME"),
		R2PublicURL:  require(v, "R2_PUBLIC_URL"),

		LogLevel: v.GetString("LOG_LEVEL"),
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	rawOrigins := require(v, "ALLOWED_ORIGINS")
	cfg.AllowedOrigins = strings.Split(rawOrigins, ",")

	return cfg
}

func (c *Config) IsDevelopment() bool { return c.AppEnv == "development" }
func (c *Config) IsProduction() bool  { return c.AppEnv == "production" }
func (c *Config) IsTest() bool        { return c.AppEnv == "test" }

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
