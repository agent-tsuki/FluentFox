package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type KanaSyncService struct {
	repo   KanaWriter
	logger *zap.Logger
}

func NewKanaSyncService(repo KanaWriter, log *zap.Logger) *KanaSyncService {
	return &KanaSyncService{repo: repo, logger: log}
}

// One transaction per file, so a bad file leaves the database as it was.
func (s *KanaSyncService) Sync(ctx context.Context, file *KanaFile) error {
	if _, err := parseKanaScript(string(file.Script)); err != nil {
		return fmt.Errorf("kana file: unknown script %q", file.Script)
	}
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		unitIDs, err := s.syncUnits(ctx, tx, file)
		if err != nil {
			return err
		}
		kanaIDs, err := s.syncKana(ctx, tx, file, unitIDs)
		if err != nil {
			return err
		}
		return s.syncLookAlikes(ctx, tx, file, kanaIDs)
	})
	if err != nil {
		return fmt.Errorf("sync %s: %w", file.Script, err)
	}
	s.logger.Info("kana synced", zap.String("script", string(file.Script)),
		zap.Int("units", len(file.Units)), zap.Int("kana", len(file.Kana)))
	return nil
}

func (s *KanaSyncService) syncUnits(ctx context.Context, tx *gorm.DB, file *KanaFile) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(file.Units))
	for i := range file.Units {
		u := &file.Units[i]
		id, err := s.repo.UpsertKanaUnit(ctx, tx, &KanaUnit{
			Script: file.Script, GroupKey: u.Group, Slug: u.Slug, Title: u.Title, Summary: u.Summary,
			Position: u.Position, YoutubeID: u.YoutubeID, VideoCredit: u.VideoCredit,
		})
		if err != nil {
			return nil, fmt.Errorf("unit %s: %w", u.Slug, err)
		}
		ids[u.Slug] = id
	}
	return ids, nil
}

func (s *KanaSyncService) syncKana(ctx context.Context, tx *gorm.DB, file *KanaFile, unitIDs map[string]uuid.UUID) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(file.Kana))
	for i := range file.Kana {
		k := &file.Kana[i]
		unitID, ok := unitIDs[k.Unit]
		if !ok {
			return nil, fmt.Errorf("kana %s: unknown unit %q", k.Slug, k.Unit)
		}
		id, err := s.repo.UpsertKana(ctx, tx, &Kana{
			Script: file.Script, Slug: k.Slug, Character: k.Character, Romaji: k.Romaji, AltRomaji: nonNil(k.AltRomaji),
			GroupKey: k.Group, UnitID: unitID, Position: k.Position, StrokeCount: k.StrokeCount,
		})
		if err != nil {
			return nil, fmt.Errorf("kana %s: %w", k.Slug, err)
		}
		ids[k.Slug] = id
	}
	return ids, nil
}

func (s *KanaSyncService) syncLookAlikes(ctx context.Context, tx *gorm.DB, file *KanaFile, kanaIDs map[string]uuid.UUID) error {
	for i := range file.Kana {
		k := &file.Kana[i]
		for _, la := range k.LookAlikes {
			otherID, ok := kanaIDs[la.Slug]
			if !ok {
				return fmt.Errorf("kana %s: unknown look-alike %q", k.Slug, la.Slug)
			}
			row := &KanaLookAlike{KanaID: kanaIDs[k.Slug], OtherKanaID: otherID, Note: la.Note}
			if err := s.repo.UpsertKanaLookAlike(ctx, tx, row); err != nil {
				return fmt.Errorf("kana %s look-alike %s: %w", k.Slug, la.Slug, err)
			}
		}
	}
	return nil
}
