// Package main — cmd/migrate/main.go
// Applies the goose SQL migrations embedded from migrations/.
// Run via: make migrate (up), make migrate-status, make migrate-down.
// Any goose command works as the first argument, e.g. `go run ./cmd/migrate down-to 0`.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/migrations"
	"github.com/fluentfox/api/pkg/database"
)

const (
	defaultCommand = "up"
	// The SQL files sit at the root of the embedded FS.
	migrationsDir = "."
)

func main() {
	cfg := config.LoadDatabase()

	log, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	// No deferred Sync: the development logger writes to stderr unbuffered and
	// Fatal syncs before it exits.

	command := defaultCommand
	var args []string
	if len(os.Args) > 1 {
		command, args = os.Args[1], os.Args[2:]
	}

	if err := run(context.Background(), cfg, command, args); err != nil {
		log.Fatal("migration failed", zap.String("command", command), zap.Error(err))
	}

	log.Info("migration complete", zap.String("command", command))
}

func run(ctx context.Context, cfg *config.DatabaseConfig, command string, args []string) error {
	db, err := database.NewDB(cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect(string(goose.DialectPostgres)); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.RunContext(ctx, command, sqlDB, migrationsDir, args...); err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}
	return nil
}
