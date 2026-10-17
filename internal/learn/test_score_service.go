package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

type TestScoreRepo interface {
	EnsureSettings(ctx context.Context, userID uuid.UUID) (LearningSettings, error)
	FindTestResult(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) (*TestResult, error)
	InsertTestResult(ctx context.Context, tx *gorm.DB, res *TestResult) error
	RecordUnitQuiz(ctx context.Context, tx *gorm.DB, p *UnitProgress) error
	UpsertKnownCards(ctx context.Context, tx *gorm.DB, cards []SRSCard) error
}

var _ TestScoreRepo = (*Repository)(nil)

// TestScoreService scores a test session when it is completed and records
// what the result unlocks.
type TestScoreService struct {
	repo    TestScoreRepo
	catalog *Catalog
	rnd     srs.RandSource
	now     func() time.Time
	logger  *zap.Logger
}

func NewTestScoreService(repo TestScoreRepo, catalog *Catalog, rnd srs.RandSource, log *zap.Logger) *TestScoreService {
	if rnd == nil {
		rnd = globalRand{}
	}
	return &TestScoreService{repo: repo, catalog: catalog, rnd: rnd, now: time.Now, logger: log}
}

// Result returns nil for a session that is not a test. The first call
// records the result; later calls return it. The caller holds the session
// row lock, so two completions cannot both record one.
func (s *TestScoreService) Result(ctx context.Context, tx *gorm.DB, session *StudySession, logs []ReviewLog) (*TestScore, error) {
	kind := common.SESSION_KIND_RULES[session.Kind].Test
	if kind == "" || session.Track == nil {
		return nil, nil
	}
	stored, err := s.repo.FindTestResult(ctx, tx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("test result: %w", err)
	}
	if stored != nil {
		return toTestScore(stored), nil
	}
	pool, err := s.catalog.Load(ctx, *session.Track)
	if err != nil {
		return nil, fmt.Errorf("test result: %w", err)
	}
	scope := ""
	if session.Scope != nil {
		scope = *session.Scope
	}
	items, scope, err := testScope(pool, kind, scope)
	if err != nil {
		return nil, fmt.Errorf("test result: %w", err)
	}
	res := scoreTest(session, kind, testTotal(items), logs, s.now().UTC())
	res.Scope = scope
	if err := s.repo.InsertTestResult(ctx, tx, &res); err != nil {
		return nil, fmt.Errorf("test result: %w", err)
	}
	if err := s.unlock(ctx, tx, &res, pool, items); err != nil {
		return nil, err
	}
	return toTestScore(&res), nil
}

// scoreTest counts the cards whose first answer was right. Unanswered
// questions count against the score, so stopping early cannot pass a test.
func scoreTest(session *StudySession, kind common.TestKind, total int, logs []ReviewLog, now time.Time) TestResult {
	correct := 0
	for _, l := range firstGradedLogs(logs) {
		if l.IsCorrect {
			correct++
		}
	}
	score := min(correct, total)
	return TestResult{
		ID: uuid.New(), UserID: session.UserID, SessionID: session.ID, Kind: kind, Track: *session.Track,
		Score: score, Total: total, ActiveMs: session.ActiveMs, TakenAt: now,
		Passed: total > 0 && float64(score)/float64(total) >= passMark(kind, *session.Track),
	}
}

func passMark(kind common.TestKind, track common.LearningTrack) float64 {
	return common.TEST_PASS_MARKS[kind][common.LEARNING_TRACK_ITEM_TYPE[track]]
}

func (s *TestScoreService) unlock(ctx context.Context, tx *gorm.DB, res *TestResult, pool *Pool, items []*Item) error {
	switch {
	case res.Kind == common.TestKindUnitQuiz:
		return s.recordUnitQuiz(ctx, tx, res, pool)
	case res.Kind == common.TestKindSkip && res.Passed:
		return s.createKnownCards(ctx, tx, res, items)
	default:
		return nil
	}
}

// A passed quiz completes the unit.
func (s *TestScoreService) recordUnitQuiz(ctx context.Context, tx *gorm.DB, res *TestResult, pool *Pool) error {
	unit, ok := pool.UnitBySlug(res.Scope)
	if !ok {
		return fmt.Errorf("record unit quiz: unit %s left the %s track", res.Scope, res.Track)
	}
	pct := 0
	if res.Total > 0 {
		pct = res.Score * int(common.PERCENT) / res.Total
	}
	p := UnitProgress{UserID: res.UserID, UnitType: unit.Type, UnitID: unit.ID, QuizBestPct: &pct}
	if res.Passed {
		completed := res.TakenAt
		p.CompletedAt = &completed
	}
	if err := s.repo.RecordUnitQuiz(ctx, tx, &p); err != nil {
		return fmt.Errorf("record unit quiz: %w", err)
	}
	return nil
}

// createKnownCards schedules every item in scope as already known. Cards
// the user is already learning keep their schedule (see UpsertKnownCards).
func (s *TestScoreService) createKnownCards(ctx context.Context, tx *gorm.DB, res *TestResult, items []*Item) error {
	settings, err := s.repo.EnsureSettings(ctx, res.UserID)
	if err != nil {
		return fmt.Errorf("create known cards: %w", err)
	}
	sched, err := newScheduler(&settings)
	if err != nil {
		return fmt.Errorf("create known cards: %w", err)
	}
	cards := make([]SRSCard, 0, testTotal(items))
	for _, item := range items {
		for _, facet := range common.SRS_ITEM_FACETS[item.Type] {
			known := sched.NewKnownCard(res.TakenAt, s.rnd)
			cards = append(cards, SRSCard{
				ID: cardID(res.UserID, item.Type, item.ID, facet), UserID: res.UserID,
				ItemType: item.Type, ItemID: item.ID, Facet: facet,
				State: known.State, Due: known.Due, Stability: known.Stability, Difficulty: known.Difficulty,
				Reps: known.Reps, Lapses: known.Lapses, LearningStep: known.LearningStep,
				ScheduledDays: known.ScheduledDays, LastReviewAt: known.LastReview, IntroducedAt: res.TakenAt,
			})
		}
	}
	if err := s.repo.UpsertKnownCards(ctx, tx, cards); err != nil {
		return fmt.Errorf("create known cards: %w", err)
	}
	return nil
}

func toTestScore(r *TestResult) *TestScore {
	return &TestScore{
		Kind: string(r.Kind), Score: r.Score, Total: r.Total, Passed: r.Passed, PassMark: passMark(r.Kind, r.Track),
	}
}
