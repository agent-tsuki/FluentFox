package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Upserts key on each table's natural key and never delete, so re-running a
// sync is safe and ids that user data points at stay stable. A new id is
// generated for the insert; RETURNING gives back the existing id on conflict.

func upsertReturningID(ctx context.Context, tx *gorm.DB, query string, args ...any) (uuid.UUID, error) {
	var id uuid.UUID
	if err := tx.WithContext(ctx).Raw(query, args...).Row().Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("scan upserted id: %w", err)
	}
	return id, nil
}

func (r *Repository) UpsertKanaUnit(ctx context.Context, tx *gorm.DB, u *KanaUnit) (uuid.UUID, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO kana_unit (id, script, group_key, slug, title, summary, position, youtube_id, video_credit, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (script, slug) DO UPDATE SET
			group_key = EXCLUDED.group_key, title = EXCLUDED.title, summary = EXCLUDED.summary,
			position = EXCLUDED.position, youtube_id = EXCLUDED.youtube_id,
			video_credit = EXCLUDED.video_credit, updated_at = now()
		RETURNING id`,
		uuid.New(), u.Script, u.GroupKey, u.Slug, u.Title, u.Summary, u.Position, u.YoutubeID, u.VideoCredit)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert kana unit %s/%s: %w", u.Script, u.Slug, err)
	}
	return id, nil
}

func (r *Repository) UpsertKana(ctx context.Context, tx *gorm.DB, k *Kana) (uuid.UUID, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO kana (id, script, slug, character, romaji, alt_romaji, group_key, unit_id, position, stroke_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (script, slug) DO UPDATE SET
			character = EXCLUDED.character, romaji = EXCLUDED.romaji, alt_romaji = EXCLUDED.alt_romaji,
			group_key = EXCLUDED.group_key, unit_id = EXCLUDED.unit_id, position = EXCLUDED.position,
			stroke_count = EXCLUDED.stroke_count, updated_at = now()
		RETURNING id`,
		uuid.New(), k.Script, k.Slug, k.Character, k.Romaji, k.AltRomaji, k.GroupKey, k.UnitID, k.Position, k.StrokeCount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert kana %s/%s: %w", k.Script, k.Slug, err)
	}
	return id, nil
}

func (r *Repository) UpsertKanaLookAlike(ctx context.Context, tx *gorm.DB, la *KanaLookAlike) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO kana_look_alike (kana_id, other_kana_id, note, created_at, updated_at)
		VALUES (?, ?, ?, now(), now())
		ON CONFLICT (kana_id, other_kana_id) DO UPDATE SET note = EXCLUDED.note, updated_at = now()`,
		la.KanaID, la.OtherKanaID, la.Note).Error
	if err != nil {
		return fmt.Errorf("upsert kana look-alike %s → %s: %w", la.KanaID, la.OtherKanaID, err)
	}
	return nil
}

func (r *Repository) UpsertKanjiGroup(ctx context.Context, tx *gorm.DB, g *KanjiGroup) (uuid.UUID, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO kanji_group (id, level, slug, title, summary, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (level, slug) DO UPDATE SET
			title = EXCLUDED.title, summary = EXCLUDED.summary, position = EXCLUDED.position, updated_at = now()
		RETURNING id`,
		uuid.New(), g.Level, g.Slug, g.Title, g.Summary, g.Position)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert kanji group %s/%s: %w", g.Level, g.Slug, err)
	}
	return id, nil
}

func (r *Repository) UpsertKanjiUnit(ctx context.Context, tx *gorm.DB, u *KanjiUnit) (uuid.UUID, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO kanji_unit (id, group_id, level, slug, title, summary, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (level, slug) DO UPDATE SET
			group_id = EXCLUDED.group_id, title = EXCLUDED.title, summary = EXCLUDED.summary,
			position = EXCLUDED.position, updated_at = now()
		RETURNING id`,
		uuid.New(), u.GroupID, u.Level, u.Slug, u.Title, u.Summary, u.Position)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert kanji unit %s/%s: %w", u.Level, u.Slug, err)
	}
	return id, nil
}

func (r *Repository) UpsertKanjiPart(ctx context.Context, tx *gorm.DB, p *KanjiPart) (uuid.UUID, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO kanji_part (id, char, name, created_at, updated_at)
		VALUES (?, ?, ?, now(), now())
		ON CONFLICT (char) DO UPDATE SET name = EXCLUDED.name, updated_at = now()
		RETURNING id`,
		uuid.New(), p.Char, p.Name)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert kanji part %s: %w", p.Char, err)
	}
	return id, nil
}

func (r *Repository) UpsertKanji(ctx context.Context, tx *gorm.DB, k *Kanji) (uuid.UUID, error) {
	id, err := upsertReturningID(ctx, tx, `
		INSERT INTO kanji (id, char, level, unit_id, position, meanings, synonyms, onyomi,
			key_reading_type, key_reading, reading_tip, mnemonic, mnemonic_draft, strokes, grade, freq,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
		ON CONFLICT (char) DO UPDATE SET
			level = EXCLUDED.level, unit_id = EXCLUDED.unit_id, position = EXCLUDED.position,
			meanings = EXCLUDED.meanings, synonyms = EXCLUDED.synonyms, onyomi = EXCLUDED.onyomi,
			key_reading_type = EXCLUDED.key_reading_type, key_reading = EXCLUDED.key_reading,
			reading_tip = EXCLUDED.reading_tip, mnemonic = EXCLUDED.mnemonic,
			mnemonic_draft = EXCLUDED.mnemonic_draft, strokes = EXCLUDED.strokes,
			grade = EXCLUDED.grade, freq = EXCLUDED.freq, updated_at = now()
		RETURNING id`,
		uuid.New(), k.Char, k.Level, k.UnitID, k.Position, k.Meanings, k.Synonyms, k.Onyomi,
		k.KeyReadingType, k.KeyReading, k.ReadingTip, k.Mnemonic, k.MnemonicDraft, k.Strokes, k.Grade, k.Freq)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert kanji %s: %w", k.Char, err)
	}
	return id, nil
}

// The plan gives kun readings and vocab no unique key, so (kanji_id, position)
// is matched with an UPDATE first and an INSERT only when nothing matched.
// That works whether or not the migration adds a unique index there.
func (r *Repository) UpsertKanjiKunReading(ctx context.Context, tx *gorm.DB, k *KanjiKunReading) error {
	res := tx.WithContext(ctx).Exec(`
		UPDATE kanji_kun_reading SET kana = ?, okurigana = ?, affix = ?, updated_at = now()
		WHERE kanji_id = ? AND position = ?`,
		k.Kana, k.Okurigana, k.Affix, k.KanjiID, k.Position)
	if res.Error != nil {
		return fmt.Errorf("update kun reading %s #%d: %w", k.KanjiID, k.Position, res.Error)
	}
	if res.RowsAffected > 0 {
		return nil
	}
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO kanji_kun_reading (id, kanji_id, kana, okurigana, affix, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, now(), now())`,
		uuid.New(), k.KanjiID, k.Kana, k.Okurigana, k.Affix, k.Position).Error
	if err != nil {
		return fmt.Errorf("insert kun reading %s #%d: %w", k.KanjiID, k.Position, err)
	}
	return nil
}

func (r *Repository) UpsertKanjiVocab(ctx context.Context, tx *gorm.DB, v *KanjiVocab) error {
	res := tx.WithContext(ctx).Exec(`
		UPDATE kanji_vocab SET word = ?, reading = ?, meaning = ?, is_key = ?, updated_at = now()
		WHERE kanji_id = ? AND position = ?`,
		v.Word, v.Reading, v.Meaning, v.IsKey, v.KanjiID, v.Position)
	if res.Error != nil {
		return fmt.Errorf("update kanji vocab %s #%d: %w", v.KanjiID, v.Position, res.Error)
	}
	if res.RowsAffected > 0 {
		return nil
	}
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO kanji_vocab (id, kanji_id, word, reading, meaning, is_key, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, now(), now())`,
		uuid.New(), v.KanjiID, v.Word, v.Reading, v.Meaning, v.IsKey, v.Position).Error
	if err != nil {
		return fmt.Errorf("insert kanji vocab %s #%d: %w", v.KanjiID, v.Position, err)
	}
	return nil
}

func (r *Repository) UpsertKanjiPartLink(ctx context.Context, tx *gorm.DB, l *KanjiPartLink) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO kanji_part_link (kanji_id, part_id, position, created_at, updated_at)
		VALUES (?, ?, ?, now(), now())
		ON CONFLICT (kanji_id, part_id) DO UPDATE SET position = EXCLUDED.position, updated_at = now()`,
		l.KanjiID, l.PartID, l.Position).Error
	if err != nil {
		return fmt.Errorf("upsert kanji part link %s → %s: %w", l.KanjiID, l.PartID, err)
	}
	return nil
}

func (r *Repository) FindKanjiIDsByChars(ctx context.Context, tx *gorm.DB, chars []string) (map[string]uuid.UUID, error) {
	var rows []Kanji
	if err := tx.WithContext(ctx).Select("id", "char").Where("char IN ?", chars).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("find kanji ids by char: %w", err)
	}
	ids := make(map[string]uuid.UUID, len(rows))
	for i := range rows {
		ids[rows[i].Char] = rows[i].ID
	}
	return ids, nil
}

// clock_timestamp() rather than now(): the API orders look-alikes by
// created_at, and now() is the same for every row of one transaction.
func (r *Repository) UpsertKanjiLookAlike(ctx context.Context, tx *gorm.DB, la *KanjiLookAlike) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO kanji_look_alike (kanji_id, other_kanji_id, note, created_at, updated_at)
		VALUES (?, ?, ?, clock_timestamp(), now())
		ON CONFLICT (kanji_id, other_kanji_id) DO UPDATE SET note = EXCLUDED.note, updated_at = now()`,
		la.KanjiID, la.OtherKanjiID, la.Note).Error
	if err != nil {
		return fmt.Errorf("upsert kanji look-alike %s → %s: %w", la.KanjiID, la.OtherKanjiID, err)
	}
	return nil
}
