package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

type ItemHistoryService struct {
	repo   ItemHistoryRepo
	pools  PoolLoader
	kana   KanaReader
	now    func() time.Time
	logger *zap.Logger
}

func NewItemHistoryService(repo ItemHistoryRepo, pools PoolLoader, kana KanaReader, log *zap.Logger) *ItemHistoryService {
	return &ItemHistoryService{repo: repo, pools: pools, kana: kana, now: time.Now, logger: log}
}

type foundItem struct {
	Pool *learn.Pool
	Item *learn.Item
}

func (s *ItemHistoryService) Get(ctx context.Context, userID uuid.UUID, in *ItemHistoryInput) (ItemHistory, error) {
	found, err := s.findItem(ctx, in)
	if err != nil {
		return ItemHistory{}, err
	}
	settings, err := s.repo.EnsureSettings(ctx, userID)
	if err != nil {
		return ItemHistory{}, fmt.Errorf("item history: %w", err)
	}
	sched, err := learn.NewSchedulerFor(&settings)
	if err != nil {
		return ItemHistory{}, fmt.Errorf("item history: %w", err)
	}
	item := found.Item
	cards, err := s.repo.ListItemCards(ctx, userID, item.Type, item.ID)
	if err != nil {
		return ItemHistory{}, fmt.Errorf("item history: %w", err)
	}
	ids := make([]uuid.UUID, len(cards))
	refs := make([]*learn.SRSCard, len(cards))
	out := ItemHistory{
		ItemType: string(item.Type), ItemID: item.ID, Track: string(found.Pool.Track), Char: item.Char,
		Cards: make([]ItemCardProgress, 0, len(cards)), Answers: []ItemAnswer{},
	}
	for i := range cards {
		ids[i], refs[i] = cards[i].ID, &cards[i]
		p, err := cardProgress(sched, &cards[i], s.now())
		if err != nil {
			return ItemHistory{}, fmt.Errorf("item history: %w", err)
		}
		out.Cards = append(out.Cards, p)
	}
	out.Level = string(itemLevel(item.Type, refs))
	if len(ids) == 0 {
		return out, nil
	}
	if out.Stats, out.Answers, err = s.answers(ctx, userID, ids); err != nil {
		return ItemHistory{}, fmt.Errorf("item history: %w", err)
	}
	return out, nil
}

func (s *ItemHistoryService) answers(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (ItemStats, []ItemAnswer, error) {
	stats, err := s.repo.ListCardAnswerStats(ctx, userID, learn.AnswerStatsFilter{CardIDs: ids})
	if err != nil {
		return ItemStats{}, nil, fmt.Errorf("load answer stats: %w", err)
	}
	var t answerTotals
	for i := range stats {
		t.add(&stats[i], nil)
	}
	logs, err := s.repo.ListRecentAnswers(ctx, userID, ids, common.ITEM_HISTORY_ANSWERS)
	if err != nil {
		return ItemStats{}, nil, fmt.Errorf("load recent answers: %w", err)
	}
	answers := make([]ItemAnswer, 0, len(logs))
	for i := range logs {
		answers = append(answers, toItemAnswer(&logs[i]))
	}
	return ItemStats{
		Answers: t.Answers, Correct: t.Correct, Accuracy: t.accuracy(),
		Reviews: t.Reviews, Retention: t.retention(), AvgAnswerMs: t.avgMs(),
	}, answers, nil
}

func cardProgress(sched *srs.Scheduler, c *learn.SRSCard, now time.Time) (ItemCardProgress, error) {
	p := ItemCardProgress{
		CardID: c.ID, Facet: string(c.Facet), State: int(c.State), Level: string(cardLevel(c)),
		Stability: c.Stability, Difficulty: c.Difficulty, Reps: c.Reps, Lapses: c.Lapses,
		LastReviewAt: c.LastReviewAt, IsLeech: c.IsLeech,
	}
	if c.State == srs.StateNew {
		return p, nil
	}
	card := c.SRS()
	r, err := sched.Retrievability(&card, now)
	if err != nil {
		return ItemCardProgress{}, fmt.Errorf("recall of card %s: %w", c.ID, err)
	}
	due := c.Due
	p.Recall, p.NextDue = &r, &due
	return p, nil
}

func toItemAnswer(l *learn.ReviewLog) ItemAnswer {
	a := ItemAnswer{
		CardID: l.CardID, QuestionType: string(l.QuestionType), AnsweredAt: l.AnsweredAt,
		Correct: l.IsCorrect, Reason: string(l.GradeReason), ActiveMs: l.ActiveMs,
	}
	if l.Rating != nil {
		label := string(learn.LabelFor(*l.Rating))
		a.Label = &label
	}
	return a
}

// findItem resolves the id among the tracks of the item type: an item id,
// a character, or (kana only) a slug. Kana slugs such as "ka" exist in both
// scripts, so such a slug needs the track.
func (s *ItemHistoryService) findItem(ctx context.Context, in *ItemHistoryInput) (foundItem, error) {
	only, err := learn.ParseOptionalTrack(in.Track)
	if err != nil {
		return foundItem{}, fmt.Errorf("find item: %w", err)
	}
	pools, err := s.pools.Pools(ctx, only)
	if err != nil {
		return foundItem{}, fmt.Errorf("find item: %w", err)
	}
	itemType := common.SRSItemType(in.Type)
	var found []foundItem
	for _, pool := range pools {
		if common.LEARNING_TRACK_ITEM_TYPE[pool.Track] != itemType {
			continue
		}
		item, ok, err := s.matchItem(ctx, pool, in.ID)
		if err != nil {
			return foundItem{}, fmt.Errorf("find item: %w", err)
		}
		if ok {
			found = append(found, foundItem{Pool: pool, Item: item})
		}
	}
	switch len(found) {
	case 0:
		return foundItem{}, exceptions.ErrProgressItemNotFound
	case 1:
		return found[0], nil
	default:
		return foundItem{}, exceptions.ErrProgressItemAmbiguous
	}
}

func (s *ItemHistoryService) matchItem(ctx context.Context, pool *learn.Pool, id string) (*learn.Item, bool, error) {
	if parsed, err := uuid.Parse(id); err == nil {
		item, ok := pool.Item(parsed)
		return item, ok, nil
	}
	if item, ok := pool.ItemByChar(id); ok {
		return item, true, nil
	}
	script, isKana := common.LEARNING_TRACK_KANA_SCRIPT[pool.Track]
	if !isKana {
		return nil, false, nil
	}
	kana, err := s.kana.ListKana(ctx, script)
	if err != nil {
		return nil, false, fmt.Errorf("match kana slug: %w", err)
	}
	for i := range kana {
		if kana[i].Slug == id {
			item, ok := pool.Item(kana[i].ID)
			return item, ok, nil
		}
	}
	return nil, false, nil
}
