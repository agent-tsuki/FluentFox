// Package main — cmd/migrate/main.go
// Runs GORM AutoMigrate to create/update database tables from model definitions.
// Run via: make migrate
package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/database"
)

func main() {
	cfg := config.LoadDatabase()

	log, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	db, err := database.NewDB(cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := dedupeRefreshTokens(db); err != nil {
		log.Fatal("refresh token dedupe failed", zap.Error(err))
	}

	log.Info("running auto-migrate")

	if err := db.AutoMigrate(
		&users.User{},
		&users.UserProfile{},
		&users.UserSettings{},
		&users.UserVerification{},
		&users.RefreshToken{},
		&users.PasswordReset{},
		&users.OAuthIdentity{},
	); err != nil {
		log.Fatal("auto-migrate failed", zap.Error(err))
	}

	log.Info("auto-migrate complete")
}

// The unique index on refresh_tokens.user_id cannot be created while older
// races left several rows for one user. Keep each user's newest row; the
// removed duplicates only sign those sessions out.
func dedupeRefreshTokens(db *gorm.DB) error {
	if !db.Migrator().HasTable(&users.RefreshToken{}) {
		return nil
	}
	return db.Exec(`
		DELETE FROM refresh_tokens older
		USING refresh_tokens newer
		WHERE older.user_id = newer.user_id
		  AND (older.updated_at, older.id) < (newer.updated_at, newer.id)`).Error
}
