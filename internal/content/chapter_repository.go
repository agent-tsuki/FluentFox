package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

// Chapter rows key on (level, slug), words on (chapter, word) and sentences
// on (chapter, sentence). Like the other upserts they never delete, so ids
// that review cards point at stay stable across syncs.

func (r *Repository) UpsertGrammarChapter(ctx context.Context, tx *gorm.DB, c *GrammarChapter) (GrammarChapter, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO grammar_chapter (id, level, slug, number, title, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (level, slug) DO UPDATE SET
			number = EXCLUDED.number, title = EXCLUDED.title, position = EXCLUDED.position, updated_at = now()
		RETURNING id`,
		uuid.New(), c.Level, c.Slug, c.Number, c.Title, c.Position)
	if err != nil {
		return GrammarChapter{}, fmt.Errorf("upsert grammar chapter %s %s: %w", c.Level, c.Slug, err)
	}
	out := *c
	out.ID = id
	return out, nil
}

func (r *Repository) UpsertChapterWord(ctx context.Context, tx *gorm.DB, w *ChapterWord) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO chapter_word (id, chapter_id, position, word, kana, romaji, meaning, hint, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (chapter_id, word) DO UPDATE SET
			position = EXCLUDED.position, kana = EXCLUDED.kana, romaji = EXCLUDED.romaji,
			meaning = EXCLUDED.meaning, hint = EXCLUDED.hint, updated_at = now()`,
		uuid.New(), w.ChapterID, w.Position, w.Word, w.Kana, w.Romaji, w.Meaning, w.Hint).Error
	if err != nil {
		return fmt.Errorf("upsert chapter word %s: %w", w.Word, err)
	}
	return nil
}

func (r *Repository) UpsertGrammarSentence(ctx context.Context, tx *gorm.DB, s *GrammarSentence) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO grammar_sentence (id, chapter_id, position, sentence, answer, distractors, also_accept, romaji,
			translation, explanation, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (chapter_id, sentence) DO UPDATE SET
			position = EXCLUDED.position, answer = EXCLUDED.answer, distractors = EXCLUDED.distractors,
			also_accept = EXCLUDED.also_accept, romaji = EXCLUDED.romaji, translation = EXCLUDED.translation,
			explanation = EXCLUDED.explanation, updated_at = now()`,
		uuid.New(), s.ChapterID, s.Position, s.Sentence, s.Answer, s.Distractors, s.AlsoAccept, s.Romaji,
		s.Translation, s.Explanation).Error
	if err != nil {
		return fmt.Errorf("upsert grammar sentence %q: %w", s.Sentence, err)
	}
	return nil
}

func (r *Repository) ListGrammarChapters(ctx context.Context, level common.JLPTLevel) ([]GrammarChapter, error) {
	var chapters []GrammarChapter
	if err := r.db.WithContext(ctx).Where("level = ?", level).Order("position").Find(&chapters).Error; err != nil {
		return nil, fmt.Errorf("list grammar chapters %s: %w", level, err)
	}
	return chapters, nil
}

func (r *Repository) ListChapterWords(ctx context.Context, level common.JLPTLevel) ([]ChapterWord, error) {
	var words []ChapterWord
	err := r.db.WithContext(ctx).
		Joins("JOIN grammar_chapter c ON c.id = chapter_word.chapter_id").
		Where("c.level = ?", level).
		Order("c.position, chapter_word.position").
		Find(&words).Error
	if err != nil {
		return nil, fmt.Errorf("list chapter words %s: %w", level, err)
	}
	return words, nil
}

func (r *Repository) ListGrammarSentences(ctx context.Context, level common.JLPTLevel) ([]GrammarSentence, error) {
	var sentences []GrammarSentence
	err := r.db.WithContext(ctx).
		Joins("JOIN grammar_chapter c ON c.id = grammar_sentence.chapter_id").
		Where("c.level = ?", level).
		Order("c.position, grammar_sentence.position").
		Find(&sentences).Error
	if err != nil {
		return nil, fmt.Errorf("list grammar sentences %s: %w", level, err)
	}
	return sentences, nil
}
