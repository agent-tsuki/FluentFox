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

type StartTestService struct {
	repo      ItemSessionRepo
	catalog   *Catalog
	questions *QuestionBuilder
	rnd       Rand
	now       func() time.Time
	logger    *zap.Logger
}

func NewStartTestService(repo ItemSessionRepo, catalog *Catalog, questions *QuestionBuilder, rnd Rand, log *zap.Logger) *StartTestService {
	if rnd == nil {
		rnd = globalRand{}
	}
	return &StartTestService{repo: repo, catalog: catalog, questions: questions, rnd: rnd, now: time.Now, logger: log}
}

// Start asks every facet of every item in scope once, in random order. Items
// the user has not learned yet are asked too: that is what a skip test is
// for. The score is worked out when the session is completed.
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
	questions := set.everyFacet(items, 0)
	s.rnd.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })

	session := newStudySession(userID, sessionKind, &track, now)
	session.Scope = &scope
	if err := s.repo.CreateSession(ctx, &session); err != nil {
		return SessionStart{}, fmt.Errorf("start test: %w", err)
	}
	return SessionStart{Session: toSessionDTO(&session), Questions: questions}, nil
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
