package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
)

type SummaryService struct {
	repo    SummaryRepo
	catalog *Catalog
	now     func() time.Time
	logger  *zap.Logger
}

func NewSummaryService(repo SummaryRepo, catalog *Catalog, log *zap.Logger) *SummaryService {
	return &SummaryService{repo: repo, catalog: catalog, now: time.Now, logger: log}
}

func (s *SummaryService) Get(ctx context.Context, userID uuid.UUID, rawTrack string) (LearnSummary, error) {
	track, err := parseOptionalTrack(rawTrack)
	if err != nil {
		return LearnSummary{}, err
	}
	now := s.now()
	l, err := loadLearner(ctx, s.repo, userID, now)
	if err != nil {
		return LearnSummary{}, fmt.Errorf("learn summary: %w", err)
	}
	dueNow, err := s.repo.CountDueCards(ctx, userID, now, track)
	if err != nil {
		return LearnSummary{}, fmt.Errorf("learn summary: %w", err)
	}
	dueToday, err := s.repo.CountDueCards(ctx, userID, l.Day.End, track)
	if err != nil {
		return LearnSummary{}, fmt.Errorf("learn summary: %w", err)
	}
	learned, err := s.repo.CountLearnedItems(ctx, userID, track)
	if err != nil {
		return LearnSummary{}, fmt.Errorf("learn summary: %w", err)
	}
	newLeft, next, err := s.newItems(ctx, &l, track)
	if err != nil {
		return LearnSummary{}, fmt.Errorf("learn summary: %w", err)
	}
	activity, err := s.repo.GetDailyActivity(ctx, userID, l.Day.LocalDate)
	if err != nil {
		return LearnSummary{}, fmt.Errorf("learn summary: %w", err)
	}
	return LearnSummary{
		DueNow:       dueNow,
		DueToday:     dueToday,
		NewLeftToday: newLeft,
		Learned:      learned,
		NextUnit:     next,
		Goal: LearnGoal{
			AnswersToday: activity.ReviewsDone,
			GoalAnswers:  l.Settings.DailyGoalAnswers,
			Met:          activity.GoalMet,
		},
	}, nil
}

// Hiragana and katakana share the daily kana limit, so the budget left by
// one track carries over to the next.
func (s *SummaryService) newItems(ctx context.Context, l *learner, track *common.LearningTrack) (int, *LearnNextUnit, error) {
	tracks := common.LEARNING_TRACKS
	if track != nil {
		tracks = []common.LearningTrack{*track}
	}
	budgets := map[common.SRSItemType]int{}
	keys := map[common.SRSItemType][]CardKey{}
	left := 0
	var next *LearnNextUnit
	for _, t := range tracks {
		itemType := common.LEARNING_TRACK_ITEM_TYPE[t]
		if _, ok := budgets[itemType]; !ok {
			budget, typeKeys, err := s.typeBudget(ctx, l, itemType)
			if err != nil {
				return 0, nil, err
			}
			budgets[itemType], keys[itemType] = budget, typeKeys
		}
		pool, err := s.catalog.Load(ctx, t)
		if err != nil {
			return 0, nil, err
		}
		pending := pendingItems(pool, keys[itemType])
		take := min(budgets[itemType], len(pending))
		left += take
		budgets[itemType] -= take
		if next == nil && len(pending) > 0 {
			next = nextUnitOf(pool, pending[0])
		}
	}
	return left, next, nil
}

func (s *SummaryService) typeBudget(ctx context.Context, l *learner, itemType common.SRSItemType) (int, []CardKey, error) {
	introduced, err := s.repo.CountItemsIntroducedSince(ctx, l.UserID, itemType, l.Day.Start)
	if err != nil {
		return 0, nil, fmt.Errorf("new item budget: %w", err)
	}
	keys, err := s.repo.ListCardKeys(ctx, l.UserID, itemType)
	if err != nil {
		return 0, nil, fmt.Errorf("new item budget: %w", err)
	}
	return max(l.newPerDay(itemType)-introduced, 0), keys, nil
}

func nextUnitOf(pool *Pool, item *Item) *LearnNextUnit {
	unit, ok := pool.Unit(item.UnitID)
	if !ok {
		return nil
	}
	return &LearnNextUnit{Type: string(unit.Type), ID: unit.ID.String(), Slug: unit.Slug, Title: unit.Title}
}
