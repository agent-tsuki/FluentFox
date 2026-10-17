package progress

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
	"github.com/fluentfox/api/internal/learn"
)

type WeakGroupsService struct {
	repo   CardStatsRepo
	pools  PoolLoader
	logger *zap.Logger
}

func NewWeakGroupsService(repo CardStatsRepo, pools PoolLoader, log *zap.Logger) *WeakGroupsService {
	return &WeakGroupsService{repo: repo, pools: pools, logger: log}
}

// groupTotals is one candidate group with the answers of its cards.
type groupTotals struct {
	Group  WeakGroup
	Totals answerTotals
}

// Get lists the units and kana groups the user misses clearly more often
// than their own average, or answers clearly slower than usual.
func (s *WeakGroupsService) Get(ctx context.Context, userID uuid.UUID, rawTrack string) (WeakGroupsResponse, error) {
	track, err := learn.ParseOptionalTrack(rawTrack)
	if err != nil {
		return WeakGroupsResponse{}, fmt.Errorf("weak groups: %w", err)
	}
	pools, err := s.pools.Pools(ctx, track)
	if err != nil {
		return WeakGroupsResponse{}, fmt.Errorf("weak groups: %w", err)
	}
	cards, err := s.repo.ListUserCards(ctx, userID)
	if err != nil {
		return WeakGroupsResponse{}, fmt.Errorf("weak groups: %w", err)
	}
	stats, err := s.repo.ListCardAnswerStats(ctx, userID, learn.AnswerStatsFilter{})
	if err != nil {
		return WeakGroupsResponse{}, fmt.Errorf("weak groups: %w", err)
	}
	usual := usualTimesOf(stats)
	var overall answerTotals
	for i := range stats {
		overall.add(&stats[i], usual)
	}
	groups := groupCards(newProgressIndex(pools, cards), totalsByCard(stats, usual))
	return WeakGroupsResponse{Groups: weakGroups(groups, overall.lapseRate())}, nil
}

// groupSet keeps the candidate groups in path order, found by key.
type groupSet struct {
	order []*groupTotals
	byKey map[string]*groupTotals
}

func (gs *groupSet) add(key string, g *WeakGroup) {
	gt := &groupTotals{Group: *g}
	gs.byKey[key] = gt
	gs.order = append(gs.order, gt)
}

// newGroupSet lists every unit of the pools and, for kana, the dakuten,
// combo and foreign groups.
func newGroupSet(pools learn.Pools) *groupSet {
	gs := &groupSet{byKey: map[string]*groupTotals{}}
	for _, pool := range pools {
		track := string(pool.Track)
		for _, u := range pool.Units {
			id := u.ID
			gs.add(id.String(), &WeakGroup{Track: track, Kind: string(common.WeakGroupUnit), UnitID: &id, Slug: u.Slug, Title: u.Title})
		}
		script, isKana := common.LEARNING_TRACK_KANA_SCRIPT[pool.Track]
		if !isKana {
			continue
		}
		for _, g := range common.WEAK_KANA_GROUPS {
			gs.add(kanaGroupKey(pool.Track, string(g)), &WeakGroup{
				Track: track, Kind: string(common.WeakGroupKanaKind), Slug: string(g), Title: content.KanaGroupTitle(script, g),
			})
		}
	}
	return gs
}

// groupCards adds each card's answers to its unit and, when there is one,
// to its kana group.
func groupCards(x *progressIndex, totals map[uuid.UUID]*answerTotals) []*groupTotals {
	gs := newGroupSet(x.pools)
	for id, t := range totals {
		ref, ok := x.byCard[id]
		if !ok {
			continue
		}
		unit, ok := ref.Pool.Unit(ref.Item.UnitID)
		if !ok {
			continue
		}
		for _, key := range []string{unit.ID.String(), kanaGroupKey(ref.Pool.Track, unit.Group)} {
			if gt, ok := gs.byKey[key]; ok {
				gt.Totals.merge(t)
			}
		}
	}
	return gs.order
}

// Unit keys are ids, so they never clash with these.
func kanaGroupKey(track common.LearningTrack, group string) string {
	return string(track) + "/" + group
}

func weakGroups(groups []*groupTotals, overallLapseRate float64) []WeakGroup {
	out := []WeakGroup{}
	for _, gt := range groups {
		t := &gt.Totals
		if t.Reviews < common.WEAK_GROUP_MIN_REVIEWS {
			continue
		}
		g := gt.Group
		g.Reviews, g.LapseRate, g.TimeRatio = t.Reviews, t.lapseRate(), t.timeRatio()
		missesMore := g.LapseRate > 0 && g.LapseRate >= common.WEAK_GROUP_LAPSE_FACTOR*overallLapseRate
		switch {
		case missesMore:
			g.Reason = fmt.Sprintf("Missed %.0f%% of reviews (your average %.0f%%)", g.LapseRate*common.PERCENT, overallLapseRate*common.PERCENT)
		case isSlow(g.TimeRatio):
			g.Reason = fmt.Sprintf("Slow: %s per answer (your usual %s)", seconds(*t.avgMs()), seconds(t.usualMs()))
		default:
			continue
		}
		g.Suggestion = groupSuggestion(&g)
		out = append(out, g)
	}
	slices.SortStableFunc(out, func(a, b WeakGroup) int {
		return cmp.Or(cmp.Compare(b.LapseRate, a.LapseRate), cmp.Compare(b.TimeRatio, a.TimeRatio))
	})
	return out
}

// A unit is practised as a unit; a kana group has no practice mode of its
// own, so it gets the weak-cards drill of its track.
func groupSuggestion(g *WeakGroup) PracticeSuggestion {
	if g.Kind == string(common.WeakGroupUnit) {
		return PracticeSuggestion{Track: g.Track, Mode: string(common.PracticeModeUnit), Unit: g.Slug}
	}
	return PracticeSuggestion{Track: g.Track, Mode: string(common.PracticeModeWeak)}
}
