package learn

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type StartPracticeService struct {
	repo      ItemSessionRepo
	catalog   *Catalog
	questions *QuestionBuilder
	rnd       Rand
	now       func() time.Time
	logger    *zap.Logger
}

func NewStartPracticeService(repo ItemSessionRepo, catalog *Catalog, questions *QuestionBuilder, rnd Rand, log *zap.Logger) *StartPracticeService {
	if rnd == nil {
		rnd = globalRand{}
	}
	return &StartPracticeService{repo: repo, catalog: catalog, questions: questions, rnd: rnd, now: time.Now, logger: log}
}

// Start builds a practice session. Its answers never change the schedule,
// except that a miss brings the card back sooner (see pullDue). The session
// has no questions when the mode finds nothing to practise.
func (s *StartPracticeService) Start(ctx context.Context, userID uuid.UUID, req *StartPracticeRequest) (SessionStart, error) {
	track, err := parseTrack(req.Track)
	if err != nil {
		return SessionStart{}, err
	}
	now := s.now()
	l, err := loadLearner(ctx, s.repo, userID, now)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start practice: %w", err)
	}
	set, err := loadItemQuestions(ctx, s.repo, s.catalog, s.questions, &l, track)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start practice: %w", err)
	}
	questions, err := s.pick(set, req, now)
	if err != nil {
		return SessionStart{}, err
	}
	session := newStudySession(userID, common.SessionKindPractice, &track, now)
	scope := practiceScope(req)
	session.Scope = &scope
	if err := s.repo.CreateSession(ctx, &session); err != nil {
		return SessionStart{}, fmt.Errorf("start practice: %w", err)
	}
	return SessionStart{Session: toSessionDTO(&session), Questions: questions}, nil
}

func (s *StartPracticeService) pick(set *itemQuestions, req *StartPracticeRequest, now time.Time) ([]Question, error) {
	size := set.learner.Settings.SessionSize
	mode := common.PracticeMode(req.Mode)
	switch mode {
	case common.PracticeModeWeak:
		return weakPractice(set, now, size)
	case common.PracticeModeUnit:
		return unitPractice(set, req.Unit, size)
	case common.PracticeModeLookalike:
		return lookalikePractice(set, req.Pair, size)
	case common.PracticeModeSpeed, common.PracticeModeMatch:
		return s.quickPractice(set, mode, size), nil
	default:
		return nil, exceptions.ErrPracticeModeInvalid
	}
}

// practiceScope records the mode, with the unit or pair it was given.
func practiceScope(req *StartPracticeRequest) string {
	switch common.PracticeMode(req.Mode) {
	case common.PracticeModeUnit:
		return req.Mode + common.PRACTICE_PAIR_SEPARATOR + req.Unit
	case common.PracticeModeLookalike:
		if req.Pair != "" {
			return req.Mode + common.PRACTICE_PAIR_SEPARATOR + req.Pair
		}
	}
	return req.Mode
}

// weakPractice asks the introduced cards with the lowest recall chance and
// the highest lapse rate first.
func weakPractice(set *itemQuestions, now time.Time, size int) ([]Question, error) {
	sched, err := newScheduler(&set.learner.Settings)
	if err != nil {
		return nil, fmt.Errorf("weak practice: %w", err)
	}
	cards := set.introduced()
	weakness := make(map[uuid.UUID]float64, len(cards))
	for _, c := range cards {
		sc := toSRSCard(c)
		r, err := sched.Retrievability(&sc, now)
		if err != nil {
			return nil, fmt.Errorf("weak practice: %w", err)
		}
		lapseRate := float64(c.Lapses) / float64(max(c.Reps, 1))
		weakness[c.ID] = common.WEAK_RECALL_WEIGHT*(1-r) + common.WEAK_LAPSE_WEIGHT*lapseRate
	}
	slices.SortStableFunc(cards, func(a, b *SRSCard) int { return cmp.Compare(weakness[b.ID], weakness[a.ID]) })
	return cardQuestions(set, cards, size, nil), nil
}

func unitPractice(set *itemQuestions, slug string, size int) ([]Question, error) {
	if slug == "" {
		return nil, exceptions.ErrPracticeUnitRequired
	}
	unit, ok := set.pool.UnitBySlug(slug)
	if !ok {
		return nil, exceptions.ErrUnitNotFound
	}
	items := set.pool.ItemsWhere(func(u Unit) bool { return u.ID == unit.ID })
	return set.everyFacet(items, size), nil
}

// lookalikePractice drills the given pair, or else the user's own mix-ups
// and then the look-alikes of items already introduced.
func lookalikePractice(set *itemQuestions, pair string, size int) ([]Question, error) {
	var items []*Item
	if pair != "" {
		var err error
		if items, err = pairItems(set.pool, pair); err != nil {
			return nil, err
		}
	} else {
		items = lookalikeItems(set)
	}
	out := make([]Question, 0, min(len(items), size))
	for _, item := range items[:min(len(items), size)] {
		out = append(out, set.buildAs(item, common.LOOKALIKE_FACETS[item.Type], common.LOOKALIKE_QUESTION_TYPES[item.Type]))
	}
	return out, nil
}

func pairItems(pool *Pool, pair string) ([]*Item, error) {
	chars := strings.Split(pair, common.PRACTICE_PAIR_SEPARATOR)
	if len(chars) != 2 || chars[0] == chars[1] {
		return nil, exceptions.ErrPracticePairInvalid
	}
	items := make([]*Item, 0, len(chars))
	for _, c := range chars {
		item, ok := pool.ItemByChar(strings.TrimSpace(c))
		if !ok {
			return nil, exceptions.ErrPracticePairInvalid
		}
		items = append(items, item)
	}
	return items, nil
}

// Mix-ups are taken in path order of the item mixed up, each with the items
// picked for it (most often first); only items of this track take part.
func lookalikeItems(set *itemQuestions) []*Item {
	var out []*Item
	seen := map[uuid.UUID]bool{}
	add := func(ids ...uuid.UUID) {
		for _, id := range ids {
			if item, ok := set.pool.Item(id); ok && !seen[id] {
				seen[id] = true
				out = append(out, item)
			}
		}
	}
	for i := range set.pool.Items {
		if chosen := set.confused[set.pool.Items[i].ID]; len(chosen) > 0 {
			add(set.pool.Items[i].ID)
			add(chosen...)
		}
	}
	for _, c := range set.introduced() {
		if item, ok := set.pool.Item(c.ItemID); ok && len(item.LookAlikes) > 0 {
			add(item.ID)
			add(item.LookAlikes...)
		}
	}
	return out
}

// quickPractice mixes introduced cards in random order. Kana are asked as
// speed or match questions; kanji as plain multiple choice.
func (s *StartPracticeService) quickPractice(set *itemQuestions, mode common.PracticeMode, size int) []Question {
	cards := set.introduced()
	s.rnd.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	set.mode = common.AnswerModeChoice
	kanaType := common.PRACTICE_QUICK_KANA_TYPES[mode]
	return cardQuestions(set, cards, size, func(item *Item) (common.QuestionType, bool) {
		return kanaType, item.Type == common.SRSItemKana
	})
}

// cardQuestions asks the given cards in order. fixedType, when set, may
// choose the question type for an item.
func cardQuestions(set *itemQuestions, cards []*SRSCard, size int, fixedType func(*Item) (common.QuestionType, bool)) []Question {
	out := make([]Question, 0, min(len(cards), size))
	for _, c := range cards {
		if len(out) == size {
			break
		}
		item, ok := set.pool.Item(c.ItemID)
		if !ok {
			continue
		}
		if fixedType != nil {
			if qt, use := fixedType(item); use {
				out = append(out, set.buildAs(item, c.Facet, qt))
				continue
			}
		}
		out = append(out, set.build(item, c.Facet))
	}
	return out
}
