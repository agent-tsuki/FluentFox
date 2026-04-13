// Package main — cmd/migrate/main.go
// Runs GORM AutoMigrate to create/update database tables from model definitions.
// Run via: make migrate
package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/database"
)

func main() {
	cfg := config.Load()

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

	log.Info("running auto-migrate")

	if err := db.AutoMigrate(
		&users.User{},
		&users.UserProfile{},
		&users.UserSettings{},
		&users.UserVerification{},
		&users.RefreshToken{},
	); err != nil {
		log.Fatal("auto-migrate failed", zap.Error(err))
	}

	log.Info("auto-migrate complete")
}
