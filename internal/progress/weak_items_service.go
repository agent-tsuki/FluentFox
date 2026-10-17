package progress

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

type WeakItemsService struct {
	repo   WeakItemsRepo
	pools  PoolLoader
	now    func() time.Time
	logger *zap.Logger
}

func NewWeakItemsService(repo WeakItemsRepo, pools PoolLoader, log *zap.Logger) *WeakItemsService {
	return &WeakItemsService{repo: repo, pools: pools, now: time.Now, logger: log}
}

// weakInputs is what scoring needs besides the card itself.
type weakInputs struct {
	sched     *srs.Scheduler
	now       time.Time
	totals    map[uuid.UUID]*answerTotals
	itemTotal map[itemKey]int
	confused  map[itemKey]*topConfusion
}

// Get ranks the introduced cards with at least WEAK_MIN_REPS reviews by
// weakness score, weakest first.
func (s *WeakItemsService) Get(ctx context.Context, userID uuid.UUID, in *WeakItemsInput) (WeakItemsResponse, error) {
	track, err := learn.ParseOptionalTrack(in.Track)
	if err != nil {
		return WeakItemsResponse{}, fmt.Errorf("weak items: %w", err)
	}
	x, inputs, err := s.load(ctx, userID, track)
	if err != nil {
		return WeakItemsResponse{}, fmt.Errorf("weak items: %w", err)
	}
	items := []WeakItem{}
	for _, ref := range x.byCard {
		if ref.Card.State == srs.StateNew || ref.Card.Reps < common.WEAK_MIN_REPS {
			continue
		}
		item, err := inputs.score(&ref)
		if err != nil {
			return WeakItemsResponse{}, fmt.Errorf("weak items: %w", err)
		}
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b WeakItem) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.CardID.String(), b.CardID.String()))
	})
	limit := in.Limit
	if limit <= 0 {
		limit = common.WEAK_ITEMS_DEFAULT_LIMIT
	}
	return WeakItemsResponse{Items: items[:min(limit, len(items))]}, nil
}

// load reads cards of the requested track(s), and the answers and mix-ups
// of every track, so the user's usual times come from all their answers.
func (s *WeakItemsService) load(ctx context.Context, userID uuid.UUID, track *common.LearningTrack) (*progressIndex, *weakInputs, error) {
	settings, err := s.repo.EnsureSettings(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("load settings: %w", err)
	}
	sched, err := learn.NewSchedulerFor(&settings)
	if err != nil {
		return nil, nil, fmt.Errorf("build scheduler: %w", err)
	}
	pools, err := s.pools.Pools(ctx, track)
	if err != nil {
		return nil, nil, fmt.Errorf("load pools: %w", err)
	}
	cards, err := s.repo.ListUserCards(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("load cards: %w", err)
	}
	stats, err := s.repo.ListCardAnswerStats(ctx, userID, learn.AnswerStatsFilter{})
	if err != nil {
		return nil, nil, fmt.Errorf("load answer stats: %w", err)
	}
	confusions, err := s.repo.ListConfusions(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("load confusions: %w", err)
	}
	x := newProgressIndex(pools, cards)
	inputs := &weakInputs{
		sched: sched, now: s.now(), totals: totalsByCard(stats, usualTimesOf(stats)),
		confused: topConfusions(pools, confusions),
	}
	inputs.itemTotal = answersByItem(x, inputs.totals)
	return x, inputs, nil
}

func (w *weakInputs) score(ref *cardRef) (WeakItem, error) {
	card := ref.Card.SRS()
	recall, err := w.sched.Retrievability(&card, w.now)
	if err != nil {
		return WeakItem{}, fmt.Errorf("score card %s: %w", ref.Card.ID, err)
	}
	totals := w.totals[ref.Card.ID]
	if totals == nil {
		totals = &answerTotals{}
	}
	key := itemKey{Type: ref.Item.Type, ID: ref.Item.ID}
	facts := weakFacts{
		Lapses: ref.Card.Lapses, Recall: recall, TimeRatio: totals.timeRatio(),
		AvgMs: totals.avgMs(), UsualMs: totals.usualMs(), Confusion: w.confused[key],
	}
	signals := weakSignals{Recall: recall, LapseRate: lapseRate(ref.Card), TimeRatio: facts.TimeRatio}
	if facts.Confusion != nil && w.itemTotal[key] > 0 {
		signals.ConfusionShare = min(float64(facts.Confusion.Times)/float64(w.itemTotal[key]), 1)
	}
	return WeakItem{
		CardID: ref.Card.ID, ItemType: string(ref.Item.Type), ItemID: ref.Item.ID, Facet: string(ref.Card.Facet),
		Track: string(ref.Pool.Track), Char: ref.Item.Char, Level: string(cardLevel(ref.Card)),
		Score: weaknessScore(signals), Recall: recall, LapseRate: signals.LapseRate, TimeRatio: signals.TimeRatio,
		ConfusionShare: signals.ConfusionShare, Reasons: weakReasons(&facts), Suggestion: weakSuggestion(ref, &facts),
	}, nil
}

// topConfusions keeps, per target item of the loaded tracks, the item picked
// instead most often. Rows come most frequent first.
func topConfusions(pools learn.Pools, rows []learn.UserConfusion) map[itemKey]*topConfusion {
	out := map[itemKey]*topConfusion{}
	for i := range rows {
		r := &rows[i]
		key := itemKey{Type: r.ItemType, ID: r.TargetItemID}
		if _, seen := out[key]; seen {
			continue
		}
		if pair, ok := confusionPair(pools, r); ok {
			out[key] = &topConfusion{Char: pair.Chosen.Char, Times: r.Times}
		}
	}
	return out
}

// answersByItem adds up the answers of every card of an item, the base of
// the confusion share.
func answersByItem(x *progressIndex, totals map[uuid.UUID]*answerTotals) map[itemKey]int {
	out := map[itemKey]int{}
	for id, t := range totals {
		if ref, ok := x.byCard[id]; ok {
			out[itemKey{Type: ref.Item.Type, ID: ref.Item.ID}] += t.Answers
		}
	}
	return out
}
