// Package cache — in-memory ContentStore implementation.
// All reads are served from Go maps under an RWMutex — zero DB calls at runtime.
// Invalidate() flushes a key and re-fetches from DB asynchronously.
package cache

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MemoryStore implements ContentStore using sync.RWMutex-protected maps.
type MemoryStore struct {
	mu         sync.RWMutex
	pool       *pgxpool.Pool
	vocab      map[string][]VocabEntry   // keyed by chapter_id
	kanji      map[string][]KanjiEntry   // keyed by jlpt_level
	chapters   map[string]*ChapterEntry  // keyed by slug
	characters map[string][]CharacterEntry // keyed by script
}

// NewMemoryStore constructs a MemoryStore backed by the given pool.
// Call WarmUp immediately after construction.
func NewMemoryStore(pool *pgxpool.Pool) *MemoryStore {
	return &MemoryStore{
		pool:       pool,
		vocab:      make(map[string][]VocabEntry),
		kanji:      make(map[string][]KanjiEntry),
		chapters:   make(map[string]*ChapterEntry),
		characters: make(map[string][]CharacterEntry),
	}
}

// WarmUp populates all cache maps from the database.
// It is called once at application startup before the HTTP server begins serving.
func (m *MemoryStore) WarmUp(ctx context.Context) error {
	if err := m.loadChapters(ctx); err != nil {
		return fmt.Errorf("cache warm up: chapters: %w", err)
	}
	if err := m.loadVocab(ctx); err != nil {
		return fmt.Errorf("cache warm up: vocab: %w", err)
	}
	if err := m.loadKanji(ctx); err != nil {
		return fmt.Errorf("cache warm up: kanji: %w", err)
	}
	if err := m.loadCharacters(ctx); err != nil {
		return fmt.Errorf("cache warm up: characters: %w", err)
	}
	return nil
}

// GetVocabByChapter returns vocab entries for the given chapter ID from memory.
func (m *MemoryStore) GetVocabByChapter(_ context.Context, chapterID string) ([]VocabEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vocab[chapterID], nil
}

// GetKanjiByLevel returns kanji entries for a JLPT level from memory.
func (m *MemoryStore) GetKanjiByLevel(_ context.Context, level string) ([]KanjiEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.kanji[level], nil
}

// GetChapter returns a chapter by slug from memory.
func (m *MemoryStore) GetChapter(_ context.Context, slug string) (*ChapterEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.chapters[slug]
	if !ok {
		return nil, nil
	}
	return ch, nil
}

// GetCharacters returns character entries for a script type from memory.
func (m *MemoryStore) GetCharacters(_ context.Context, script string) ([]CharacterEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.characters[script], nil
}

// Invalidate removes the entry for key and re-fetches it from the database.
func (m *MemoryStore) Invalidate(ctx context.Context, key string) error {
	m.mu.Lock()
	delete(m.vocab, key)
	delete(m.kanji, key)
	delete(m.chapters, key)
	delete(m.characters, key)
	m.mu.Unlock()
	return m.WarmUp(ctx)
}

func (m *MemoryStore) loadChapters(ctx context.Context) error {
	rows, err := m.pool.Query(ctx,
		`SELECT id, slug, title, jlpt_level, order_index FROM chapters ORDER BY order_index`)
	if err != nil {
		return fmt.Errorf("cache: query chapters: %w", err)
	}
	defer rows.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	for rows.Next() {
		var c ChapterEntry
		if err := rows.Scan(&c.ID, &c.Slug, &c.Title, &c.JLPTLevel, &c.OrderIndex); err != nil {
			return fmt.Errorf("cache: scan chapter: %w", err)
		}
		m.chapters[c.Slug] = &c
	}
	return rows.Err()
}

func (m *MemoryStore) loadVocab(ctx context.Context) error {
	rows, err := m.pool.Query(ctx,
		`SELECT id, chapter_id, word, reading, meaning, jlpt_level FROM vocabulary`)
	if err != nil {
		return fmt.Errorf("cache: query vocab: %w", err)
	}
	defer rows.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	for rows.Next() {
		var v VocabEntry
		var chapterID string
		if err := rows.Scan(&v.ID, &chapterID, &v.Word, &v.Reading, &v.Meaning, &v.JLPTLevel); err != nil {
			return fmt.Errorf("cache: scan vocab: %w", err)
		}
		m.vocab[chapterID] = append(m.vocab[chapterID], v)
	}
	return rows.Err()
}

func (m *MemoryStore) loadKanji(ctx context.Context) error {
	rows, err := m.pool.Query(ctx,
		`SELECT id, character, meaning, on_reading, kun_reading, jlpt_level FROM kanji_entries`)
	if err != nil {
		return fmt.Errorf("cache: query kanji: %w", err)
	}
	defer rows.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	for rows.Next() {
		var k KanjiEntry
		if err := rows.Scan(&k.ID, &k.Character, &k.Meaning, &k.OnReading, &k.KunReading, &k.JLPTLevel); err != nil {
			return fmt.Errorf("cache: scan kanji: %w", err)
		}
		m.kanji[k.JLPTLevel] = append(m.kanji[k.JLPTLevel], k)
	}
	return rows.Err()
}

func (m *MemoryStore) loadCharacters(ctx context.Context) error {
	rows, err := m.pool.Query(ctx,
		`SELECT id, script, romaji, character FROM characters ORDER BY script, id`)
	if err != nil {
		return fmt.Errorf("cache: query characters: %w", err)
	}
	defer rows.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	for rows.Next() {
		var c CharacterEntry
		if err := rows.Scan(&c.ID, &c.Script, &c.Romaji, &c.Character); err != nil {
			return fmt.Errorf("cache: scan character: %w", err)
		}
		m.characters[c.Script] = append(m.characters[c.Script], c)
	}
	return rows.Err()
}
