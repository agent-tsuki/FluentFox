// Package main — cmd/sync-content/main.go.
// CLI tool that reads the JSON files in content/ and upserts them into PostgreSQL.
// Run via: make sync-content
// It must never be imported by the HTTP server — it is a standalone binary.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/content"
	"github.com/fluentfox/api/pkg/database"
	"go.uber.org/zap"
)

const contentDir = "content"

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

	repo := content.NewRepository(db)
	if err := syncAll(context.Background(), repo, log); err != nil {
		log.Fatal("content sync failed", zap.Error(err))
	}

	log.Info("content sync complete")
}

// Kana first: nothing depends on it. Kanji files run in name order (n1 ... n5),
// and a look-alike pointing at a level synced later links on the next run.
func syncAll(ctx context.Context, repo *content.Repository, log *zap.Logger) error {
	if err := syncFiles(ctx, filepath.Join(contentDir, "kana"), content.NewKanaSyncService(repo, log).Sync); err != nil {
		return err
	}
	return syncFiles(ctx, filepath.Join(contentDir, "kanji"), content.NewKanjiSyncService(repo, log).Sync)
}

func syncFiles[T any](ctx context.Context, dir string, sync func(context.Context, *T) error) error {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return fmt.Errorf("list %s: %w", dir, err)
	}
	for _, path := range paths {
		var file T
		if err := readJSON(path, &file); err != nil {
			return err
		}
		if err := sync(ctx, &file); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from a Glob over the content directory
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
