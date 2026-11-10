package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

// Passages key on (chapter, position) and lines on (passage, position).
// Nothing points at them (they are not review cards), so a sync also drops
// the passages and lines past the end of the file's lists.

func (r *Repository) ChapterIDBySlug(ctx context.Context, tx *gorm.DB, level common.JLPTLevel, slug string) (uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.WithContext(ctx).Table("grammar_chapter").Where("level = ? AND slug = ?", level, slug).Pluck("id", &ids).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("find grammar chapter %s %s: %w", level, slug, err)
	}
	if len(ids) == 0 {
		return uuid.Nil, nil
	}
	return ids[0], nil
}

func (r *Repository) UpsertChapterPassage(ctx context.Context, tx *gorm.DB, p *ChapterPassage) (uuid.UUID, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO chapter_passage (id, chapter_id, position, kind, title, speakers, speaker_voices, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (chapter_id, position) DO UPDATE SET
			kind = EXCLUDED.kind, title = EXCLUDED.title, speakers = EXCLUDED.speakers,
			speaker_voices = EXCLUDED.speaker_voices, updated_at = now()
		RETURNING id`,
		uuid.New(), p.ChapterID, p.Position, p.Kind, p.Title, p.Speakers, p.SpeakerVoices)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert chapter passage %d: %w", p.Position, err)
	}
	return id, nil
}

func (r *Repository) UpsertPassageLine(ctx context.Context, tx *gorm.DB, l *ChapterPassageLine) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO chapter_passage_line (id, passage_id, position, speaker, text, translation, say, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (passage_id, position) DO UPDATE SET
			speaker = EXCLUDED.speaker, text = EXCLUDED.text, translation = EXCLUDED.translation,
			say = EXCLUDED.say, updated_at = now()`,
		uuid.New(), l.PassageID, l.Position, l.Speaker, l.Text, l.Translation, l.Say).Error
	if err != nil {
		return fmt.Errorf("upsert passage line %d: %w", l.Position, err)
	}
	return nil
}

// TrimPassages drops a chapter's passages after the first keep.
func (r *Repository) TrimPassages(ctx context.Context, tx *gorm.DB, chapterID uuid.UUID, keep int) error {
	if err := tx.WithContext(ctx).Exec(`DELETE FROM chapter_passage WHERE chapter_id = ? AND position > ?`, chapterID, keep).Error; err != nil {
		return fmt.Errorf("trim chapter passages: %w", err)
	}
	return nil
}

func (r *Repository) UpsertChapterExercise(ctx context.Context, tx *gorm.DB, e *ChapterExercise) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO chapter_exercise (id, chapter_id, position, kind, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?::jsonb, now(), now())
		ON CONFLICT (chapter_id, position) DO UPDATE SET
			kind = EXCLUDED.kind, data = EXCLUDED.data, updated_at = now()`,
		uuid.New(), e.ChapterID, e.Position, e.Kind, string(e.Data)).Error
	if err != nil {
		return fmt.Errorf("upsert chapter exercise %d: %w", e.Position, err)
	}
	return nil
}

func (r *Repository) TrimChapterExercises(ctx context.Context, tx *gorm.DB, chapterID uuid.UUID, keep int) error {
	if err := tx.WithContext(ctx).Exec(`DELETE FROM chapter_exercise WHERE chapter_id = ? AND position > ?`, chapterID, keep).Error; err != nil {
		return fmt.Errorf("trim chapter exercises: %w", err)
	}
	return nil
}

func (r *Repository) ListChapterExercises(ctx context.Context, chapterID uuid.UUID) ([]ChapterExercise, error) {
	var items []ChapterExercise
	if err := r.db.WithContext(ctx).Where("chapter_id = ?", chapterID).Order("position").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list chapter exercises: %w", err)
	}
	return items, nil
}

// TrimPassageLines drops a passage's lines after the first keep.
func (r *Repository) TrimPassageLines(ctx context.Context, tx *gorm.DB, passageID uuid.UUID, keep int) error {
	if err := tx.WithContext(ctx).Exec(`DELETE FROM chapter_passage_line WHERE passage_id = ? AND position > ?`, passageID, keep).Error; err != nil {
		return fmt.Errorf("trim passage lines: %w", err)
	}
	return nil
}

func (r *Repository) ListChapterPassages(ctx context.Context, chapterID uuid.UUID) ([]ChapterPassage, error) {
	var passages []ChapterPassage
	if err := r.db.WithContext(ctx).Where("chapter_id = ?", chapterID).Order("position").Find(&passages).Error; err != nil {
		return nil, fmt.Errorf("list chapter passages: %w", err)
	}
	return passages, nil
}

func (r *Repository) ListPassageLines(ctx context.Context, chapterID uuid.UUID) ([]ChapterPassageLine, error) {
	var lines []ChapterPassageLine
	err := r.db.WithContext(ctx).
		Joins("JOIN chapter_passage p ON p.id = chapter_passage_line.passage_id").
		Where("p.chapter_id = ?", chapterID).
		Order("p.position, chapter_passage_line.position").
		Find(&lines).Error
	if err != nil {
		return nil, fmt.Errorf("list passage lines: %w", err)
	}
	return lines, nil
}
