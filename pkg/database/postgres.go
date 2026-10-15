// Package database provides the GORM database connection.
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB creates a *gorm.DB connected to PostgreSQL using the provided DSN.
// Connection pool limits are configured from the caller.
func NewDB(dsn string, maxConns, minConns int32) (*gorm.DB, error) {
	// Skip GORM's per-write BEGIN/COMMIT (two extra round trips to Neon); single
	// statements are atomic and repo.Transaction is unaffected. PrepareStmt stays
	// off for the Neon pooler, and the Ping below replaces GORM's own ping.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
		DisableAutomaticPing:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(int(maxConns))
	sqlDB.SetMaxIdleConns(int(minConns))
	sqlDB.SetConnMaxLifetime(1 * time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	return db, nil
}
