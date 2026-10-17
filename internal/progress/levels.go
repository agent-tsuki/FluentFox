package progress

import (
	"math"
	"slices"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

// A card a test or practice created is still new (state 0): it was never
// scheduled, so it says nothing about memory.
func cardLevel(c *learn.SRSCard) common.MasteryLevel {
	switch {
	case c.State == srs.StateNew:
		return common.MasteryNew
	case c.State == srs.StateLearning || c.State == srs.StateRelearning || c.Stability < common.FAMILIAR_MIN_STABILITY_DAYS:
		return common.MasteryLearning
	case c.Stability < common.STRONG_MIN_STABILITY_DAYS:
		return common.MasteryFamiliar
	case c.Stability < common.MASTERED_MIN_STABILITY_DAYS:
		return common.MasteryStrong
	default:
		return common.MasteryMastered
	}
}

func lowerLevel(a, b common.MasteryLevel) common.MasteryLevel {
	if slices.Index(common.MASTERY_LEVELS, b) < slices.Index(common.MASTERY_LEVELS, a) {
		return b
	}
	return a
}

// itemLevel is the lowest level of the item's facets; a facet without a
// card counts as new, so a kanji is mastered only when both cards are.
func itemLevel(itemType common.SRSItemType, cards []*learn.SRSCard) common.MasteryLevel {
	byFacet := make(map[common.CardFacet]*learn.SRSCard, len(cards))
	for _, c := range cards {
		byFacet[c.Facet] = c
	}
	level := common.MasteryMastered
	for _, f := range common.SRS_ITEM_FACETS[itemType] {
		c, ok := byFacet[f]
		if !ok {
			return common.MasteryNew
		}
		level = lowerLevel(level, cardLevel(c))
	}
	return level
}

type itemKey struct {
	Type common.SRSItemType
	ID   uuid.UUID
}

// cardRef places a card in its track and item.
type cardRef struct {
	Card *learn.SRSCard
	Pool *learn.Pool
	Item *learn.Item
}

// progressIndex joins the user's cards to the loaded content. Cards of items
// outside the loaded pools are left out.
type progressIndex struct {
	pools  learn.Pools
	byCard map[uuid.UUID]cardRef
	byItem map[itemKey][]*learn.SRSCard
}

func newProgressIndex(pools learn.Pools, cards []learn.SRSCard) *progressIndex {
	x := &progressIndex{pools: pools, byCard: make(map[uuid.UUID]cardRef, len(cards)), byItem: map[itemKey][]*learn.SRSCard{}}
	for i := range cards {
		c := &cards[i]
		pool, item, ok := pools.Find(c.ItemType, c.ItemID)
		if !ok {
			continue
		}
		x.byCard[c.ID] = cardRef{Card: c, Pool: pool, Item: item}
		key := itemKey{Type: c.ItemType, ID: c.ItemID}
		x.byItem[key] = append(x.byItem[key], c)
	}
	return x
}

func (x *progressIndex) itemLevel(item *learn.Item) common.MasteryLevel {
	return itemLevel(item.Type, x.byItem[itemKey{Type: item.Type, ID: item.ID}])
}

// levelCounts counts the pool's items per level.
func (x *progressIndex) levelCounts(pool *learn.Pool) ItemsPerLevel {
	var out ItemsPerLevel
	for i := range pool.Items {
		out.add(x.itemLevel(&pool.Items[i]))
	}
	return out
}

// trackProgress counts the track's items in units that pass keep, and how
// many of them have a level that passes reached.
func (x *progressIndex) trackProgress(track common.LearningTrack, keep func(learn.Unit) bool, reached func(common.MasteryLevel) bool) (int, int) {
	for _, pool := range x.pools {
		if pool.Track != track {
			continue
		}
		items := pool.ItemsWhere(keep)
		n := 0
		for _, item := range items {
			if reached(x.itemLevel(item)) {
				n++
			}
		}
		return n, len(items)
	}
	return 0, 0
}

func (l *ItemsPerLevel) add(level common.MasteryLevel) {
	switch level {
	case common.MasteryNew:
		l.New++
	case common.MasteryLearning:
		l.Learning++
	case common.MasteryFamiliar:
		l.Familiar++
	case common.MasteryStrong:
		l.Strong++
	case common.MasteryMastered:
		l.Mastered++
	}
}

func isLearned(l common.MasteryLevel) bool  { return l != common.MasteryNew }
func isMastered(l common.MasteryLevel) bool { return l == common.MasteryMastered }
func anyUnit(learn.Unit) bool               { return true }

func percentOf(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return roundPercent(float64(part) / float64(whole))
}

// roundPercent turns a 0–1 share into a percentage with one decimal.
func roundPercent(share float64) float64 {
	return math.Round(share*common.PERCENT*common.PERCENT_ROUNDING) / common.PERCENT_ROUNDING
}
