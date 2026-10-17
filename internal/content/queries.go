package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	if err := r.db.WithContext(ctx).Transaction(fn); err != nil {
		return fmt.Errorf("content transaction: %w", err)
	}
	return nil
}

func (r *Repository) ListKanaUnits(ctx context.Context, script common.KanaScript) ([]KanaUnit, error) {
	var units []KanaUnit
	if err := r.db.WithContext(ctx).Where("script = ?", script).Order("position").Find(&units).Error; err != nil {
		return nil, fmt.Errorf("list kana units for %s: %w", script, err)
	}
	return units, nil
}

func (r *Repository) ListKana(ctx context.Context, script common.KanaScript) ([]Kana, error) {
	var kana []Kana
	if err := r.db.WithContext(ctx).Where("script = ?", script).Order("position").Find(&kana).Error; err != nil {
		return nil, fmt.Errorf("list kana for %s: %w", script, err)
	}
	return kana, nil
}

func (r *Repository) ListKanaLookAlikes(ctx context.Context, script common.KanaScript) ([]KanaLookAlikeRow, error) {
	var rows []KanaLookAlikeRow
	err := r.db.WithContext(ctx).
		Table("kana_look_alike AS la").
		Select("la.kana_id, o.slug AS other_slug, o.character AS other_char, la.note").
		Joins("JOIN kana k ON k.id = la.kana_id").
		Joins("JOIN kana o ON o.id = la.other_kana_id").
		Where("k.script = ?", script).
		Order("o.position").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list kana look-alikes for %s: %w", script, err)
	}
	return rows, nil
}

func (r *Repository) ListKanjiGroups(ctx context.Context, level common.JLPTLevel) ([]KanjiGroup, error) {
	var groups []KanjiGroup
	if err := r.db.WithContext(ctx).Where("level = ?", level).Order("position").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("list kanji groups for %s: %w", level, err)
	}
	return groups, nil
}

func (r *Repository) ListKanjiUnits(ctx context.Context, level common.JLPTLevel) ([]KanjiUnit, error) {
	var units []KanjiUnit
	if err := r.db.WithContext(ctx).Where("level = ?", level).Order("position").Find(&units).Error; err != nil {
		return nil, fmt.Errorf("list kanji units for %s: %w", level, err)
	}
	return units, nil
}

func (r *Repository) ListKanji(ctx context.Context, level common.JLPTLevel) ([]Kanji, error) {
	var kanji []Kanji
	if err := r.db.WithContext(ctx).Where("level = ?", level).Order("position").Find(&kanji).Error; err != nil {
		return nil, fmt.Errorf("list kanji for %s: %w", level, err)
	}
	return kanji, nil
}

func (r *Repository) GetKanjiByChar(ctx context.Context, char string) (Kanji, error) {
	var kanji Kanji
	if err := r.db.WithContext(ctx).Where("char = ?", char).First(&kanji).Error; err != nil {
		return Kanji{}, fmt.Errorf("get kanji %s: %w", char, err)
	}
	return kanji, nil
}

func (r *Repository) GetKanjiUnit(ctx context.Context, unitID uuid.UUID) (KanjiUnit, error) {
	var unit KanjiUnit
	if err := r.db.WithContext(ctx).Where("id = ?", unitID).First(&unit).Error; err != nil {
		return KanjiUnit{}, fmt.Errorf("get kanji unit %s: %w", unitID, err)
	}
	return unit, nil
}

func (r *Repository) ListKanjiKunReadings(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiKunReading, error) {
	var readings []KanjiKunReading
	if err := r.db.WithContext(ctx).Where("kanji_id IN ?", kanjiIDs).Order("position").Find(&readings).Error; err != nil {
		return nil, fmt.Errorf("list kanji kun readings: %w", err)
	}
	return readings, nil
}

func (r *Repository) ListKanjiParts(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiPartRow, error) {
	var rows []KanjiPartRow
	err := r.db.WithContext(ctx).
		Table("kanji_part_link AS l").
		Select("l.kanji_id, p.char, p.name").
		Joins("JOIN kanji_part p ON p.id = l.part_id").
		Where("l.kanji_id IN ?", kanjiIDs).
		Order("l.position").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list kanji parts: %w", err)
	}
	return rows, nil
}

func (r *Repository) ListKanjiVocab(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiVocab, error) {
	var vocab []KanjiVocab
	if err := r.db.WithContext(ctx).Where("kanji_id IN ?", kanjiIDs).Order("position").Find(&vocab).Error; err != nil {
		return nil, fmt.Errorf("list kanji vocab: %w", err)
	}
	return vocab, nil
}

// The table has no position column, so insertion order stands in for the
// order the content file lists them in.
func (r *Repository) ListKanjiLookAlikes(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiLookAlikeRow, error) {
	var rows []KanjiLookAlikeRow
	err := r.db.WithContext(ctx).
		Table("kanji_look_alike AS la").
		Select("la.kanji_id, o.char AS other_char, la.note").
		Joins("JOIN kanji o ON o.id = la.other_kanji_id").
		Where("la.kanji_id IN ?", kanjiIDs).
		Order("la.created_at, o.char").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list kanji look-alikes: %w", err)
	}
	return rows, nil
}
