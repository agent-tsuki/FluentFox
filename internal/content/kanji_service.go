package content

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type KanjiContentService struct {
	repo           KanjiReader
	strokesBaseURL string
	logger         *zap.Logger
}

func NewKanjiContentService(repo KanjiReader, strokesBaseURL string, log *zap.Logger) *KanjiContentService {
	return &KanjiContentService{repo: repo, strokesBaseURL: strokesBaseURL, logger: log}
}

// kanjiChildren holds the per-kanji rows of the child tables, keyed by kanji id.
type kanjiChildren struct {
	kun        map[uuid.UUID][]KanjiKunReading
	parts      map[uuid.UUID][]KanjiPartRow
	vocab      map[uuid.UUID][]KanjiVocab
	lookAlikes map[uuid.UUID][]KanjiLookAlikeRow
}

func (s *KanjiContentService) GetLevelIndex(ctx context.Context, rawLevel string) (KanjiLevelIndex, error) {
	level, err := parseKanjiLevel(rawLevel)
	if err != nil {
		return KanjiLevelIndex{}, err
	}
	groups, err := s.repo.ListKanjiGroups(ctx, level)
	if err != nil {
		return KanjiLevelIndex{}, fmt.Errorf("get kanji index: %w", err)
	}
	units, err := s.repo.ListKanjiUnits(ctx, level)
	if err != nil {
		return KanjiLevelIndex{}, fmt.Errorf("get kanji index: %w", err)
	}
	kanji, err := s.repo.ListKanji(ctx, level)
	if err != nil {
		return KanjiLevelIndex{}, fmt.Errorf("get kanji index: %w", err)
	}
	children, err := s.loadChildren(ctx, kanjiIDs(kanji), false)
	if err != nil {
		return KanjiLevelIndex{}, fmt.Errorf("get kanji index: %w", err)
	}

	unitSlugs := kanjiUnitSlugs(units)
	rows := make([]KanjiIndexRow, 0, len(kanji))
	for i := range kanji {
		rows = append(rows, buildKanjiIndexRow(&kanji[i], unitSlugs[kanji[i].UnitID], children))
	}
	return KanjiLevelIndex{
		Level:          levelID(level),
		StrokesBaseURL: s.strokesBaseURL,
		Units:          buildKanjiUnits(units, kanji, children),
		Groups:         buildKanjiGroups(groups, units),
		Rows:           rows,
	}, nil
}

func (s *KanjiContentService) GetLevelDetail(ctx context.Context, rawLevel string) (KanjiLevelDetail, error) {
	level, err := parseKanjiLevel(rawLevel)
	if err != nil {
		return KanjiLevelDetail{}, err
	}
	units, err := s.repo.ListKanjiUnits(ctx, level)
	if err != nil {
		return KanjiLevelDetail{}, fmt.Errorf("get kanji detail: %w", err)
	}
	kanji, err := s.repo.ListKanji(ctx, level)
	if err != nil {
		return KanjiLevelDetail{}, fmt.Errorf("get kanji detail: %w", err)
	}
	children, err := s.loadChildren(ctx, kanjiIDs(kanji), true)
	if err != nil {
		return KanjiLevelDetail{}, fmt.Errorf("get kanji detail: %w", err)
	}

	unitSlugs := kanjiUnitSlugs(units)
	entries := make([]KanjiEntry, 0, len(kanji))
	for i := range kanji {
		entries = append(entries, buildKanjiEntry(&kanji[i], unitSlugs[kanji[i].UnitID], children))
	}
	return KanjiLevelDetail{Level: levelID(level), StrokesBaseURL: s.strokesBaseURL, Entries: entries}, nil
}

// Only kanji of a level that is served are found, so this route never leaks
// content the level routes would refuse.
func (s *KanjiContentService) GetEntry(ctx context.Context, char string) (KanjiEntryResponse, error) {
	kanji, err := s.repo.GetKanjiByChar(ctx, char)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return KanjiEntryResponse{}, exceptions.ErrKanjiNotFound
	}
	if err != nil {
		return KanjiEntryResponse{}, fmt.Errorf("get kanji entry: %w", err)
	}
	if !slices.Contains(common.KANJI_CONTENT_LEVELS, kanji.Level) {
		return KanjiEntryResponse{}, exceptions.ErrKanjiNotFound
	}
	unit, err := s.repo.GetKanjiUnit(ctx, kanji.UnitID)
	if err != nil {
		return KanjiEntryResponse{}, fmt.Errorf("get kanji entry: %w", err)
	}
	children, err := s.loadChildren(ctx, []uuid.UUID{kanji.ID}, true)
	if err != nil {
		return KanjiEntryResponse{}, fmt.Errorf("get kanji entry: %w", err)
	}
	return KanjiEntryResponse{
		KanjiEntry:     buildKanjiEntry(&kanji, unit.Slug, children),
		StrokesBaseURL: s.strokesBaseURL,
	}, nil
}

// URLs use lower-case ids (n5); the database enum is upper-case (N5).
func parseKanjiLevel(raw string) (common.JLPTLevel, error) {
	level := common.JLPTLevel(strings.ToUpper(raw))
	if raw != strings.ToLower(raw) || !slices.Contains(common.KANJI_CONTENT_LEVELS, level) {
		return "", exceptions.ErrKanjiLevelNotFound
	}
	return level, nil
}

func levelID(level common.JLPTLevel) string {
	return strings.ToLower(string(level))
}

// Vocab and look-alikes are only needed for full entries, so the index skips them.
func (s *KanjiContentService) loadChildren(ctx context.Context, ids []uuid.UUID, full bool) (kanjiChildren, error) {
	children := kanjiChildren{}
	if len(ids) == 0 {
		return children, nil
	}
	kun, err := s.repo.ListKanjiKunReadings(ctx, ids)
	if err != nil {
		return children, fmt.Errorf("load kun readings: %w", err)
	}
	parts, err := s.repo.ListKanjiParts(ctx, ids)
	if err != nil {
		return children, fmt.Errorf("load parts: %w", err)
	}
	children.kun = groupByKanji(kun, func(r *KanjiKunReading) uuid.UUID { return r.KanjiID })
	children.parts = groupByKanji(parts, func(r *KanjiPartRow) uuid.UUID { return r.KanjiID })
	if !full {
		return children, nil
	}
	vocab, err := s.repo.ListKanjiVocab(ctx, ids)
	if err != nil {
		return children, fmt.Errorf("load vocab: %w", err)
	}
	lookAlikes, err := s.repo.ListKanjiLookAlikes(ctx, ids)
	if err != nil {
		return children, fmt.Errorf("load look-alikes: %w", err)
	}
	children.vocab = groupByKanji(vocab, func(r *KanjiVocab) uuid.UUID { return r.KanjiID })
	children.lookAlikes = groupByKanji(lookAlikes, func(r *KanjiLookAlikeRow) uuid.UUID { return r.KanjiID })
	return children, nil
}

func groupByKanji[T any](rows []T, key func(*T) uuid.UUID) map[uuid.UUID][]T {
	out := make(map[uuid.UUID][]T)
	for i := range rows {
		k := key(&rows[i])
		out[k] = append(out[k], rows[i])
	}
	return out
}

func kanjiIDs(kanji []Kanji) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(kanji))
	for i := range kanji {
		ids = append(ids, kanji[i].ID)
	}
	return ids
}

func kanjiUnitSlugs(units []KanjiUnit) map[uuid.UUID]string {
	slugs := make(map[uuid.UUID]string, len(units))
	for i := range units {
		slugs[units[i].ID] = units[i].Slug
	}
	return slugs
}
