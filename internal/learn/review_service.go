package learn

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

type StartReviewService struct {
	repo      ReviewRepo
	catalog   *Catalog
	questions *QuestionBuilder
	now       func() time.Time
	logger    *zap.Logger
}

func NewStartReviewService(repo ReviewRepo, catalog *Catalog, questions *QuestionBuilder, log *zap.Logger) *StartReviewService {
	return &StartReviewService{repo: repo, catalog: catalog, questions: questions, now: time.Now, logger: log}
}

// Start builds one question per due card, capped by the session size and by
// the review answers left today. The session has no questions when nothing
// is due or the daily review limit is used up.
func (s *StartReviewService) Start(ctx context.Context, userID uuid.UUID, req *StartReviewRequest) (SessionStart, error) {
	track, err := parseOptionalTrack(req.Track)
	if err != nil {
		return SessionStart{}, err
	}
	now := s.now()
	l, err := loadLearner(ctx, s.repo, userID, now)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start review: %w", err)
	}
	cards, err := s.pickCards(ctx, &l, track, now)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start review: %w", err)
	}
	questions, err := s.buildQuestions(ctx, &l, track, cards)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start review: %w", err)
	}
	session := newStudySession(userID, common.SessionKindReview, track, now)
	if err := s.repo.CreateSession(ctx, &session); err != nil {
		return SessionStart{}, fmt.Errorf("start review: %w", err)
	}
	return SessionStart{Session: toSessionDTO(&session), Questions: questions}, nil
}

func (s *StartReviewService) pickCards(ctx context.Context, l *learner, track *common.LearningTrack, now time.Time) ([]SRSCard, error) {
	done, err := s.repo.CountReviewAnswersSince(ctx, l.UserID, l.Day.Start)
	if err != nil {
		return nil, fmt.Errorf("pick review cards: %w", err)
	}
	limit := min(l.Settings.SessionSize, max(l.Settings.MaxReviewsPerDay-done, 0))
	if limit == 0 {
		return nil, nil
	}
	cards, err := s.repo.ListDueCards(ctx, l.UserID, now, track)
	if err != nil {
		return nil, fmt.Errorf("pick review cards: %w", err)
	}
	sched, err := newScheduler(&l.Settings)
	if err != nil {
		return nil, err
	}
	if err := orderReviewCards(sched, cards, now); err != nil {
		return nil, err
	}
	return cards[:min(limit, len(cards))], nil
}

// orderReviewCards puts relearning cards first, then the lowest recall
// chance, then the longest overdue.
func orderReviewCards(sched *srs.Scheduler, cards []SRSCard, now time.Time) error {
	recall := make(map[uuid.UUID]float64, len(cards))
	for i := range cards {
		c := toSRSCard(&cards[i])
		r, err := sched.Retrievability(&c, now)
		if err != nil {
			return fmt.Errorf("order review cards: %w", err)
		}
		recall[cards[i].ID] = r
	}
	slices.SortStableFunc(cards, func(a, b SRSCard) int {
		aRelearn, bRelearn := a.State == srs.StateRelearning, b.State == srs.StateRelearning
		if aRelearn != bRelearn {
			if aRelearn {
				return -1
			}
			return 1
		}
		if c := cmp.Compare(recall[a.ID], recall[b.ID]); c != 0 {
			return c
		}
		return a.Due.Compare(b.Due)
	})
	return nil
}

func (s *StartReviewService) buildQuestions(ctx context.Context, l *learner, track *common.LearningTrack, cards []SRSCard) ([]Question, error) {
	questions := make([]Question, 0, len(cards))
	if len(cards) == 0 {
		return questions, nil
	}
	pools, err := s.catalog.Pools(ctx, track)
	if err != nil {
		return nil, err
	}
	confusions, err := s.repo.ListConfusions(ctx, l.UserID)
	if err != nil {
		return nil, fmt.Errorf("build review questions: %w", err)
	}
	confused := groupConfusions(confusions)
	for i := range cards {
		c := &cards[i]
		pool, item, ok := pools.Find(c.ItemType, c.ItemID)
		if !ok {
			s.logger.Warn("due card has no content item", zap.String("card_id", c.ID.String()))
			continue
		}
		questions = append(questions, s.questions.Build(&QuestionInput{
			Card:     QuestionCard{ID: c.ID, State: c.State, Stability: c.Stability, Facet: c.Facet, IsLeech: c.IsLeech},
			Item:     item,
			Pool:     pool,
			Confused: confused[item.ID],
			Mode:     l.Settings.AnswerMode,
		}))
	}
	return questions, nil
}

// Rows come most frequent first, so each list keeps that order.
func groupConfusions(rows []UserConfusion) map[uuid.UUID][]uuid.UUID {
	out := make(map[uuid.UUID][]uuid.UUID)
	for i := range rows {
		out[rows[i].TargetItemID] = append(out[rows[i].TargetItemID], rows[i].ChosenItemID)
	}
	return out
}

func toSRSCard(c *SRSCard) srs.Card {
	return srs.Card{
		State:         c.State,
		Due:           c.Due,
		Stability:     c.Stability,
		Difficulty:    c.Difficulty,
		Reps:          c.Reps,
		Lapses:        c.Lapses,
		LearningStep:  c.LearningStep,
		ScheduledDays: c.ScheduledDays,
		LastReview:    c.LastReviewAt,
	}
}
