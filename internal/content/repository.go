package content

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

// KanaReader is what KanaContentService needs; *Repository satisfies it.
type KanaReader interface {
	ListKanaUnits(ctx context.Context, script common.KanaScript) ([]KanaUnit, error)
	ListKana(ctx context.Context, script common.KanaScript) ([]Kana, error)
	ListKanaLookAlikes(ctx context.Context, script common.KanaScript) ([]KanaLookAlikeRow, error)
}

// KanjiReader is what KanjiContentService needs; *Repository satisfies it.
type KanjiReader interface {
	ListKanjiGroups(ctx context.Context, level common.JLPTLevel) ([]KanjiGroup, error)
	ListKanjiUnits(ctx context.Context, level common.JLPTLevel) ([]KanjiUnit, error)
	ListKanji(ctx context.Context, level common.JLPTLevel) ([]Kanji, error)
	GetKanjiByChar(ctx context.Context, char string) (Kanji, error)
	GetKanjiUnit(ctx context.Context, unitID uuid.UUID) (KanjiUnit, error)
	ListKanjiKunReadings(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiKunReading, error)
	ListKanjiParts(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiPartRow, error)
	ListKanjiVocab(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiVocab, error)
	ListKanjiLookAlikes(ctx context.Context, kanjiIDs []uuid.UUID) ([]KanjiLookAlikeRow, error)
}

// KanaWriter is what KanaSyncService needs. Upserts return the stored row id,
// which is the existing id when the natural key already exists.
type KanaWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	UpsertKanaUnit(ctx context.Context, tx *gorm.DB, unit *KanaUnit) (uuid.UUID, error)
	UpsertKana(ctx context.Context, tx *gorm.DB, kana *Kana) (uuid.UUID, error)
	UpsertKanaLookAlike(ctx context.Context, tx *gorm.DB, lookAlike *KanaLookAlike) error
}

// KanjiWriter is what KanjiSyncService needs.
type KanjiWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	UpsertKanjiGroup(ctx context.Context, tx *gorm.DB, group *KanjiGroup) (uuid.UUID, error)
	UpsertKanjiUnit(ctx context.Context, tx *gorm.DB, unit *KanjiUnit) (uuid.UUID, error)
	UpsertKanjiPart(ctx context.Context, tx *gorm.DB, part *KanjiPart) (uuid.UUID, error)
	UpsertKanji(ctx context.Context, tx *gorm.DB, kanji *Kanji) (uuid.UUID, error)
	UpsertKanjiKunReading(ctx context.Context, tx *gorm.DB, reading *KanjiKunReading) error
	UpsertKanjiVocab(ctx context.Context, tx *gorm.DB, vocab *KanjiVocab) error
	UpsertKanjiPartLink(ctx context.Context, tx *gorm.DB, link *KanjiPartLink) error
	FindKanjiIDsByChars(ctx context.Context, tx *gorm.DB, chars []string) (map[string]uuid.UUID, error)
	UpsertKanjiLookAlike(ctx context.Context, tx *gorm.DB, lookAlike *KanjiLookAlike) error
}
