package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type TestSessionRepo interface {
	ItemSessionRepo
	ListUnfinishedTests(ctx context.Context, userID uuid.UUID, kind common.SessionKind, track common.LearningTrack, scope string) ([]uuid.UUID, error)
}

var _ TestSessionRepo = (*Repository)(nil)

// SessionCloser completes a session and scores it (CompleteSessionService).
type SessionCloser interface {
	Complete(ctx context.Context, userID, sessionID uuid.UUID) (SessionResult, error)
}

type StartTestService struct {
	repo      TestSessionRepo
	catalog   *Catalog
	questions *QuestionBuilder
	closer    SessionCloser
	rnd       Rand
	now       func() time.Time
	logger    *zap.Logger
}

func NewStartTestService(repo TestSessionRepo, catalog *Catalog, questions *QuestionBuilder, closer SessionCloser, rnd Rand, log *zap.Logger) *StartTestService {
	if rnd == nil {
		rnd = globalRand{}
	}
	return &StartTestService{repo: repo, catalog: catalog, questions: questions, closer: closer, rnd: rnd, now: time.Now, logger: log}
}

// Start asks every facet of every item in scope once, in random order; a
// kana unit quiz instead asks each kana several times, both ways round (see
// kanaQuizPlan), and a grammar chapter's test mixes its kinds of exercise.
// Items the user has not learned yet are asked too: that is
// what a skip test is for. The score is worked out when the session is
// completed.
func (s *StartTestService) Start(ctx context.Context, userID uuid.UUID, req *StartTestRequest) (SessionStart, error) {
	kind := common.TestKind(req.Kind)
	sessionKind, ok := common.TEST_KIND_SESSION_KIND[kind]
	if !ok {
		return SessionStart{}, exceptions.ErrTestKindInvalid
	}
	track, err := parseTrack(req.Track)
	if err != nil {
		return SessionStart{}, err
	}
	now := s.now()
	l, err := loadLearner(ctx, s.repo, userID, now)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start test: %w", err)
	}
	set, err := loadItemQuestions(ctx, s.repo, s.catalog, s.questions, &l, track)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start test: %w", err)
	}
	items, scope, err := testScope(set.pool, kind, req.Scope)
	if err != nil {
		return SessionStart{}, err
	}
	var questions []Question
	if isKanaUnitQuiz(kind, items) {
		questions = set.kanaQuizQuestions(items, s.rnd)
	} else {
		questions = set.everyFacet(items, 0)
		if kind == common.TestKindUnitQuiz {
			if session := set.chapterSession(items, scope, s.rnd); session != nil {
				questions = session
			}
		}
		s.rnd.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
	}

	s.closeUnfinished(ctx, userID, sessionKind, track, scope)

	session := newStudySession(userID, sessionKind, &track, now)
	session.Scope = &scope
	session.QuestionPlan = planOf(questions)
	if err := s.repo.CreateSession(ctx, &session); err != nil {
		return SessionStart{}, fmt.Errorf("start test: %w", err)
	}
	return SessionStart{Session: toSessionDTO(&session), Questions: questions}, nil
}

// closeUnfinished scores the user's earlier attempts at the same test that
// were left part way (the tab closed or reloaded), so leaving a test that is
// going badly still counts as an attempt. Attempts with no answers are left
// alone. A failure is logged rather than blocking the new test.
func (s *StartTestService) closeUnfinished(ctx context.Context, userID uuid.UUID, kind common.SessionKind, track common.LearningTrack, scope string) {
	ids, err := s.repo.ListUnfinishedTests(ctx, userID, kind, track, scope)
	if err != nil {
		s.logger.Warn("list unfinished tests", zap.Error(err))
		return
	}
	for _, id := range ids {
		if _, err := s.closer.Complete(ctx, userID, id); err != nil {
			s.logger.Warn("close unfinished test", zap.String("session_id", id.String()), zap.Error(err))
		}
	}
}

// testScope resolves a test's scope to its items in path order, and returns
// the scope as stored: the unit slug, the group, or TEST_SCOPE_ALL.
func testScope(pool *Pool, kind common.TestKind, raw string) ([]*Item, string, error) {
	everyUnit := func(Unit) bool { return true }
	switch kind {
	case common.TestKindUnitQuiz:
		unit, ok := pool.UnitBySlug(raw)
		if !ok {
			return nil, "", exceptions.ErrUnitNotFound
		}
		return pool.ItemsWhere(func(u Unit) bool { return u.ID == unit.ID }), unit.Slug, nil
	case common.TestKindSkip:
		if raw == "" || raw == common.TEST_SCOPE_ALL {
			return pool.ItemsWhere(everyUnit), common.TEST_SCOPE_ALL, nil
		}
		items := pool.ItemsWhere(func(u Unit) bool { return u.Group == raw })
		if len(items) == 0 {
			return nil, "", exceptions.ErrTestScopeInvalid
		}
		return items, raw, nil
	case common.TestKindFinal:
		return pool.ItemsWhere(everyUnit), common.TEST_SCOPE_ALL, nil
	default:
		return nil, "", exceptions.ErrTestKindInvalid
	}
}

// testTotal is the number of questions a test over items asks.
func testTotal(items []*Item) int {
	n := 0
	for _, item := range items {
		n += len(common.SRS_ITEM_FACETS[item.Type])
	}
	return n
}

// testQuestionCount is the number of questions a test asks, which is what
// it is scored out of.
func testQuestionCount(kind common.TestKind, items []*Item) int {
	if isKanaUnitQuiz(kind, items) {
		return kanaUnitQuizTotal(len(items))
	}
	return testTotal(items)
}
