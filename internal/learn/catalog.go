package learn

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
)

// Item is one kana or kanji as the learning code needs it. Readings hold the
// on'yomi, the kun stems and the stems with okurigana.
type Item struct {
	Type       common.SRSItemType
	ID         uuid.UUID
	Char       string
	UnitID     uuid.UUID
	Romaji     string
	AltRomaji  []string
	Meanings   []string
	Synonyms   []string
	Readings   []string
	KeyReading string
	LookAlikes []uuid.UUID
}

type Unit struct {
	Type  common.UnitType
	ID    uuid.UUID
	Slug  string
	Title string
}

// Pool is the content of one track in path order.
type Pool struct {
	Track common.LearningTrack
	Items []Item
	Units []Unit
	byID  map[uuid.UUID]int
}

func newPool(track common.LearningTrack, units []Unit, items []Item) *Pool {
	p := &Pool{Track: track, Items: items, Units: units, byID: make(map[uuid.UUID]int, len(items))}
	for i := range items {
		p.byID[items[i].ID] = i
	}
	return p
}

func (p *Pool) Item(id uuid.UUID) (*Item, bool) {
	i, ok := p.byID[id]
	if !ok {
		return nil, false
	}
	return &p.Items[i], true
}

func (p *Pool) Unit(id uuid.UUID) (Unit, bool) {
	for _, u := range p.Units {
		if u.ID == id {
			return u, true
		}
	}
	return Unit{}, false
}

// ContentReader is what Catalog needs; *content.Repository satisfies it.
type ContentReader interface {
	ListKanaUnits(ctx context.Context, script common.KanaScript) ([]content.KanaUnit, error)
	ListKana(ctx context.Context, script common.KanaScript) ([]content.Kana, error)
	ListKanaLookAlikes(ctx context.Context, script common.KanaScript) ([]content.KanaLookAlikeRow, error)
	ListKanjiUnits(ctx context.Context, level common.JLPTLevel) ([]content.KanjiUnit, error)
	ListKanji(ctx context.Context, level common.JLPTLevel) ([]content.Kanji, error)
	ListKanjiKunReadings(ctx context.Context, kanjiIDs []uuid.UUID) ([]content.KanjiKunReading, error)
	ListKanjiLookAlikes(ctx context.Context, kanjiIDs []uuid.UUID) ([]content.KanjiLookAlikeRow, error)
}

// Catalog loads track content for the learning services.
type Catalog struct {
	content ContentReader
}

func NewCatalog(reader ContentReader) *Catalog {
	return &Catalog{content: reader}
}

func (c *Catalog) Load(ctx context.Context, track common.LearningTrack) (*Pool, error) {
	if script, ok := common.LEARNING_TRACK_KANA_SCRIPT[track]; ok {
		return c.loadKana(ctx, track, script)
	}
	return c.loadKanji(ctx, track, common.LEARNING_TRACK_KANJI_LEVEL[track])
}

// Pools loads one track, or every track when track is nil.
func (c *Catalog) Pools(ctx context.Context, track *common.LearningTrack) (Pools, error) {
	tracks := common.LEARNING_TRACKS
	if track != nil {
		tracks = []common.LearningTrack{*track}
	}
	pools := make(Pools, 0, len(tracks))
	for _, t := range tracks {
		p, err := c.Load(ctx, t)
		if err != nil {
			return nil, err
		}
		pools = append(pools, p)
	}
	return pools, nil
}

// Pools finds an item in whichever loaded track holds it.
type Pools []*Pool

func (ps Pools) Find(itemType common.SRSItemType, id uuid.UUID) (*Pool, *Item, bool) {
	for _, p := range ps {
		if common.LEARNING_TRACK_ITEM_TYPE[p.Track] != itemType {
			continue
		}
		if item, ok := p.Item(id); ok {
			return p, item, true
		}
	}
	return nil, nil, false
}

func (c *Catalog) loadKana(ctx context.Context, track common.LearningTrack, script common.KanaScript) (*Pool, error) {
	units, err := c.content.ListKanaUnits(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}
	kana, err := c.content.ListKana(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}
	lookAlikes, err := c.content.ListKanaLookAlikes(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}

	unitOrder := make(map[uuid.UUID]int, len(units))
	poolUnits := make([]Unit, 0, len(units))
	for i := range units {
		unitOrder[units[i].ID] = units[i].Position
		poolUnits = append(poolUnits, Unit{Type: common.UnitTypeKana, ID: units[i].ID, Slug: units[i].Slug, Title: units[i].Title})
	}
	idBySlug := make(map[string]uuid.UUID, len(kana))
	for i := range kana {
		idBySlug[kana[i].Slug] = kana[i].ID
	}
	similar := make(map[uuid.UUID][]uuid.UUID)
	for _, la := range lookAlikes {
		if other, ok := idBySlug[la.OtherSlug]; ok {
			similar[la.KanaID] = append(similar[la.KanaID], other)
		}
	}

	slices.SortStableFunc(kana, func(a, b content.Kana) int {
		if d := unitOrder[a.UnitID] - unitOrder[b.UnitID]; d != 0 {
			return d
		}
		return a.Position - b.Position
	})
	items := make([]Item, 0, len(kana))
	for i := range kana {
		k := &kana[i]
		items = append(items, Item{
			Type: common.SRSItemKana, ID: k.ID, Char: k.Character, UnitID: k.UnitID,
			Romaji: k.Romaji, AltRomaji: k.AltRomaji, LookAlikes: similar[k.ID],
		})
	}
	return newPool(track, poolUnits, items), nil
}

func (c *Catalog) loadKanji(ctx context.Context, track common.LearningTrack, level common.JLPTLevel) (*Pool, error) {
	units, err := c.content.ListKanjiUnits(ctx, level)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}
	kanji, err := c.content.ListKanji(ctx, level)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}
	ids := make([]uuid.UUID, len(kanji))
	for i := range kanji {
		ids[i] = kanji[i].ID
	}
	kun, err := c.content.ListKanjiKunReadings(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}
	lookAlikes, err := c.content.ListKanjiLookAlikes(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}

	unitOrder := make(map[uuid.UUID]int, len(units))
	poolUnits := make([]Unit, 0, len(units))
	for i := range units {
		unitOrder[units[i].ID] = units[i].Position
		poolUnits = append(poolUnits, Unit{Type: common.UnitTypeKanji, ID: units[i].ID, Slug: units[i].Slug, Title: units[i].Title})
	}
	slices.SortStableFunc(kanji, func(a, b content.Kanji) int {
		if d := unitOrder[a.UnitID] - unitOrder[b.UnitID]; d != 0 {
			return d
		}
		return a.Position - b.Position
	})
	return newPool(track, poolUnits, buildKanjiItems(kanji, kun, lookAlikes)), nil
}

func buildKanjiItems(kanji []content.Kanji, kun []content.KanjiKunReading, lookAlikes []content.KanjiLookAlikeRow) []Item {
	kunByKanji := make(map[uuid.UUID][]string)
	for i := range kun {
		r := &kun[i]
		kunByKanji[r.KanjiID] = append(kunByKanji[r.KanjiID], r.Kana)
		if r.Okurigana != nil && *r.Okurigana != "" {
			kunByKanji[r.KanjiID] = append(kunByKanji[r.KanjiID], r.Kana+*r.Okurigana)
		}
	}
	idByChar := make(map[string]uuid.UUID, len(kanji))
	for i := range kanji {
		idByChar[kanji[i].Char] = kanji[i].ID
	}
	similar := make(map[uuid.UUID][]uuid.UUID)
	for _, la := range lookAlikes {
		if other, ok := idByChar[la.OtherChar]; ok {
			similar[la.KanjiID] = append(similar[la.KanjiID], other)
		}
	}

	items := make([]Item, 0, len(kanji))
	for i := range kanji {
		k := &kanji[i]
		readings := append(slices.Clone([]string(k.Onyomi)), kunByKanji[k.ID]...)
		items = append(items, Item{
			Type: common.SRSItemKanji, ID: k.ID, Char: k.Char, UnitID: k.UnitID,
			Meanings: k.Meanings, Synonyms: k.Synonyms, Readings: uniqueStrings(readings),
			KeyReading: k.KeyReading, LookAlikes: similar[k.ID],
		})
	}
	return items
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
