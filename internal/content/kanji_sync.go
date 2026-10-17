package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type KanjiSyncService struct {
	repo   KanjiWriter
	logger *zap.Logger
}

func NewKanjiSyncService(repo KanjiWriter, log *zap.Logger) *KanjiSyncService {
	return &KanjiSyncService{repo: repo, logger: log}
}

// One transaction per file, so a bad file leaves the database as it was.
func (s *KanjiSyncService) Sync(ctx context.Context, file *KanjiFile) error {
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		unitIDs, err := s.syncGroupsAndUnits(ctx, tx, file)
		if err != nil {
			return err
		}
		partIDs, err := s.syncParts(ctx, tx, file)
		if err != nil {
			return err
		}
		for i := range file.Entries {
			if err := s.syncEntry(ctx, tx, file, &file.Entries[i], unitIDs, partIDs); err != nil {
				return err
			}
		}
		return s.syncLookAlikes(ctx, tx, file)
	})
	if err != nil {
		return fmt.Errorf("sync kanji %s: %w", file.Level, err)
	}
	s.logger.Info("kanji synced", zap.String("level", string(file.Level)),
		zap.Int("units", len(file.Units)), zap.Int("kanji", len(file.Entries)))
	return nil
}

func (s *KanjiSyncService) syncGroupsAndUnits(ctx context.Context, tx *gorm.DB, file *KanjiFile) (map[string]uuid.UUID, error) {
	groupIDs := make(map[string]uuid.UUID, len(file.Groups))
	for _, g := range file.Groups {
		id, err := s.repo.UpsertKanjiGroup(ctx, tx, &KanjiGroup{
			Level: file.Level, Slug: g.Slug, Title: g.Title, Summary: g.Summary, Position: g.Position,
		})
		if err != nil {
			return nil, fmt.Errorf("group %s: %w", g.Slug, err)
		}
		groupIDs[g.Slug] = id
	}
	unitIDs := make(map[string]uuid.UUID, len(file.Units))
	for _, u := range file.Units {
		groupID, ok := groupIDs[u.Group]
		if !ok {
			return nil, fmt.Errorf("kanji unit %s: unknown group %q", u.Slug, u.Group)
		}
		id, err := s.repo.UpsertKanjiUnit(ctx, tx, &KanjiUnit{
			GroupID: groupID, Level: file.Level, Slug: u.Slug, Title: u.Title, Summary: u.Summary, Position: u.Position,
		})
		if err != nil {
			return nil, fmt.Errorf("unit %s: %w", u.Slug, err)
		}
		unitIDs[u.Slug] = id
	}
	return unitIDs, nil
}

func (s *KanjiSyncService) syncParts(ctx context.Context, tx *gorm.DB, file *KanjiFile) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(file.Parts))
	for _, p := range file.Parts {
		id, err := s.repo.UpsertKanjiPart(ctx, tx, &KanjiPart{Char: p.Char, Name: p.Name})
		if err != nil {
			return nil, fmt.Errorf("part %s: %w", p.Char, err)
		}
		ids[p.Char] = id
	}
	return ids, nil
}

func (s *KanjiSyncService) syncEntry(
	ctx context.Context, tx *gorm.DB, file *KanjiFile, e *KanjiFileEntry,
	unitIDs, partIDs map[string]uuid.UUID,
) error {
	unitID, ok := unitIDs[e.Unit]
	if !ok {
		return fmt.Errorf("kanji %s: unknown unit %q", e.Char, e.Unit)
	}
	kanjiID, err := s.repo.UpsertKanji(ctx, tx, kanjiFromFile(file, e, unitID))
	if err != nil {
		return fmt.Errorf("kanji %s: %w", e.Char, err)
	}
	for i, k := range e.Kun {
		reading := &KanjiKunReading{KanjiID: kanjiID, Kana: k.Kana, Okurigana: k.Okurigana, Affix: k.Affix, Position: i + 1}
		if err := s.repo.UpsertKanjiKunReading(ctx, tx, reading); err != nil {
			return fmt.Errorf("kanji %s kun reading: %w", e.Char, err)
		}
	}
	for i, v := range e.Vocab {
		vocab := &KanjiVocab{KanjiID: kanjiID, Word: v.Word, Reading: v.Reading, Meaning: v.Meaning, IsKey: v.IsKey, Position: i + 1}
		if err := s.repo.UpsertKanjiVocab(ctx, tx, vocab); err != nil {
			return fmt.Errorf("kanji %s vocab: %w", e.Char, err)
		}
	}
	for i, char := range e.Parts {
		partID, ok := partIDs[char]
		if !ok {
			return fmt.Errorf("kanji %s: unknown part %q", e.Char, char)
		}
		if err := s.repo.UpsertKanjiPartLink(ctx, tx, &KanjiPartLink{KanjiID: kanjiID, PartID: partID, Position: i + 1}); err != nil {
			return fmt.Errorf("kanji %s part: %w", e.Char, err)
		}
	}
	return nil
}

func kanjiFromFile(file *KanjiFile, e *KanjiFileEntry, unitID uuid.UUID) *Kanji {
	k := &Kanji{
		Char: e.Char, Level: file.Level, UnitID: unitID, Position: e.Position,
		Meanings: nonNil(e.Meanings), Synonyms: nonNil(e.Synonyms), Onyomi: nonNil(e.Onyomi),
		KeyReadingType: e.KeyReading.Type, KeyReading: e.KeyReading.Kana, ReadingTip: e.ReadingTip,
		MnemonicDraft: true, Strokes: e.Strokes, Grade: e.Grade, Freq: e.Freq,
	}
	if e.Mnemonic != nil {
		k.Mnemonic = &e.Mnemonic.Text
		k.MnemonicDraft = e.Mnemonic.Draft
	}
	return k
}

// Look-alikes may point at kanji of another level. Those only link once that
// level is synced, so a missing one is logged and skipped rather than failing
// the file; re-running the sync later adds the link.
func (s *KanjiSyncService) syncLookAlikes(ctx context.Context, tx *gorm.DB, file *KanjiFile) error {
	chars := make([]string, 0, len(file.Entries))
	for i := range file.Entries {
		chars = append(chars, file.Entries[i].Char)
		for _, la := range file.Entries[i].LookAlikes {
			chars = append(chars, la.Char)
		}
	}
	ids, err := s.repo.FindKanjiIDsByChars(ctx, tx, chars)
	if err != nil {
		return fmt.Errorf("look-alikes: %w", err)
	}
	var skipped []string
	for i := range file.Entries {
		e := &file.Entries[i]
		for _, la := range e.LookAlikes {
			otherID, ok := ids[la.Char]
			if !ok {
				skipped = append(skipped, e.Char+"→"+la.Char)
				continue
			}
			row := &KanjiLookAlike{KanjiID: ids[e.Char], OtherKanjiID: otherID, Note: la.Note}
			if err := s.repo.UpsertKanjiLookAlike(ctx, tx, row); err != nil {
				return fmt.Errorf("kanji %s look-alike %s: %w", e.Char, la.Char, err)
			}
		}
	}
	if len(skipped) > 0 {
		s.logger.Warn("look-alike kanji not in database yet, skipped", zap.Strings("pairs", skipped))
	}
	return nil
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
