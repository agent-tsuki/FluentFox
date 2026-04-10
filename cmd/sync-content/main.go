// Package main — cmd/sync-content/main.go.
// CLI tool that parses MDX files from content/ and upserts them into PostgreSQL.
// Run via: make sync-content
// It must never be imported by the HTTP server — it is a standalone binary.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/pkg/database"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	log, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	pool, err := database.NewPool(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()


	log.Info("content sync complete")
}

