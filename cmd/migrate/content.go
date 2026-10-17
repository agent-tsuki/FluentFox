package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"

	"go.uber.org/zap"
	"gorm.io/gorm"

	contentfiles "github.com/fluentfox/api/content"
	"github.com/fluentfox/api/internal/content"
)

const (
	kanaContentDir  = "kana"
	kanjiContentDir = "kanji"
)

// syncContent upserts the embedded course content after the schema is up.
// Running it on every deploy keeps production content in step with the repo;
// it never deletes rows, so it is safe to repeat.
func syncContent(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	repo := content.NewRepository(db)
	// Kana first: nothing depends on it. Kanji files run in name order (n1 ... n5),
	// and a look-alike pointing at a level synced later links on the next run.
	if err := syncFiles(ctx, kanaContentDir, content.NewKanaSyncService(repo, log).Sync); err != nil {
		return err
	}
	return syncFiles(ctx, kanjiContentDir, content.NewKanjiSyncService(repo, log).Sync)
}

func syncFiles[T any](ctx context.Context, dir string, sync func(context.Context, *T) error) error {
	paths, err := fs.Glob(contentfiles.FS, path.Join(dir, "*.json"))
	if err != nil {
		return fmt.Errorf("list %s: %w", dir, err)
	}
	for _, p := range paths {
		var file T
		if err := readJSON(p, &file); err != nil {
			return err
		}
		if err := sync(ctx, &file); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
}

func readJSON(p string, v any) error {
	data, err := fs.ReadFile(contentfiles.FS, p)
	if err != nil {
		return fmt.Errorf("read %s: %w", p, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", p, err)
	}
	return nil
}
