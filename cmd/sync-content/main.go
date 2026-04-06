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
	"github.com/fluentfox/api/internal/mdxparser"
	"github.com/fluentfox/api/pkg/database"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/pgxpool"
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

	contentDir := "content/grammar"
	if len(os.Args) > 1 {
		contentDir = os.Args[1]
	}

	parser := mdxparser.New()
	chapters, err := parser.ParseDirectory(contentDir)
	if err != nil {
		log.Fatal("parse directory failed", zap.String("dir", contentDir), zap.Error(err))
	}

	log.Info("parsed chapters", zap.Int("count", len(chapters)))

	for _, ch := range chapters {
		if err := upsertChapter(context.Background(), pool, ch, log); err != nil {
			log.Error("upsert chapter failed",
				zap.String("slug", ch.Frontmatter.Slug), zap.Error(err))
		} else {
			log.Info("synced chapter", zap.String("slug", ch.Frontmatter.Slug))
		}
	}

	log.Info("content sync complete")
}

func upsertChapter(ctx context.Context, pool *pgxpool.Pool, ch *mdxparser.ParsedChapter, log *zap.Logger) error {
	// Placeholder: actual upsert logic would use pool.Exec to INSERT ... ON CONFLICT DO UPDATE
	// for chapters, concepts, vocabulary, and examples.
	_ = ch
	_ = log
	return nil
}
