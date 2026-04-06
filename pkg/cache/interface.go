// Package cache defines the ContentStore interface and its implementations.
// Services use the interface only — they never depend on a concrete type.
package cache

import "context"

// VocabEntry is a lightweight vocab item held in the cache.
type VocabEntry struct {
	ID       string
	Word     string
	Reading  string
	Meaning  string
	JLPTLevel string
}

// KanjiEntry is a lightweight kanji item held in the cache.
type KanjiEntry struct {
	ID        string
	Character string
	Meaning   string
	OnReading string
	KunReading string
	JLPTLevel  string
}

// ChapterEntry holds the cached representation of a grammar chapter.
type ChapterEntry struct {
	ID        string
	Slug      string
	Title     string
	JLPTLevel string
	OrderIndex int
}

// CharacterEntry is a kana character held in the cache.
type CharacterEntry struct {
	ID       string
	Script   string
	Romaji   string
	Character string
}

// ContentStore is the interface every cache implementation must satisfy.
// All read methods return data from memory — no DB calls at runtime.
type ContentStore interface {
	// GetVocabByChapter returns all vocabulary items for a given chapter ID.
	GetVocabByChapter(ctx context.Context, chapterID string) ([]VocabEntry, error)

	// GetKanjiByLevel returns all kanji entries for a given JLPT level.
	GetKanjiByLevel(ctx context.Context, level string) ([]KanjiEntry, error)

	// GetChapter returns a single chapter by its slug.
	GetChapter(ctx context.Context, slug string) (*ChapterEntry, error)

	// GetCharacters returns all characters for a given script (hiragana/katakana).
	GetCharacters(ctx context.Context, script string) ([]CharacterEntry, error)

	// Invalidate removes the cache entry for the given key and re-fetches from DB.
	Invalidate(ctx context.Context, key string) error

	// WarmUp populates the cache from the database. Called once at startup.
	WarmUp(ctx context.Context) error
}
