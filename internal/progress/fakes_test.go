package progress

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

var (
	testUserID = uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	// 10:00 UTC; with day_starts_at_hour 4 the UTC study day began at 04:00.
	testNow = time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)

	unitA   = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	unitGa  = uuid.MustParse("10000000-0000-0000-0000-000000000002")
	unitKa  = uuid.MustParse("10000000-0000-0000-0000-000000000003")
	unitK01 = uuid.MustParse("10000000-0000-0000-0000-000000000004")

	hiraA  = uuid.MustParse("20000000-0000-0000-0000-000000000001")
	hiraI  = uuid.MustParse("20000000-0000-0000-0000-000000000002")
	hiraGa = uuid.MustParse("20000000-0000-0000-0000-000000000003")
	kataA  = uuid.MustParse("20000000-0000-0000-0000-000000000004")
	kanji1 = uuid.MustParse("20000000-0000-0000-0000-000000000005")
	kanji2 = uuid.MustParse("20000000-0000-0000-0000-000000000006")
)

// testPools: hiragana あ い (row a, basic) and が (row ga, dakuten);
// katakana ア (row a); N5 kanji 一 二 (unit u01).
func testPools() learn.Pools {
	hira := learn.NewPool(common.LearningTrackHiragana,
		[]learn.Unit{
			{Type: common.UnitTypeKana, ID: unitA, Slug: "a", Title: "あ row", Group: string(common.KanaGroupBasic)},
			{Type: common.UnitTypeKana, ID: unitGa, Slug: "ga", Title: "が row", Group: string(common.KanaGroupDakuten)},
		},
		[]learn.Item{
			{Type: common.SRSItemKana, ID: hiraA, Char: "あ", UnitID: unitA, Romaji: "a"},
			{Type: common.SRSItemKana, ID: hiraI, Char: "い", UnitID: unitA, Romaji: "i"},
			{Type: common.SRSItemKana, ID: hiraGa, Char: "が", UnitID: unitGa, Romaji: "ga"},
		})
	kata := learn.NewPool(common.LearningTrackKatakana,
		[]learn.Unit{{Type: common.UnitTypeKana, ID: unitKa, Slug: "a", Title: "ア row", Group: string(common.KanaGroupBasic)}},
		[]learn.Item{{Type: common.SRSItemKana, ID: kataA, Char: "ア", UnitID: unitKa, Romaji: "a"}})
	kanji := learn.NewPool(common.LearningTrackKanjiN5,
		[]learn.Unit{{Type: common.UnitTypeKanji, ID: unitK01, Slug: "u01", Title: "Numbers", Group: "numbers"}},
		[]learn.Item{
			{Type: common.SRSItemKanji, ID: kanji1, Char: "一", UnitID: unitK01},
			{Type: common.SRSItemKanji, ID: kanji2, Char: "二", UnitID: unitK01},
		})
	return learn.Pools{hira, kata, kanji}
}

type fakePools struct {
	pools learn.Pools
	err   error
}

func (f *fakePools) Pools(_ context.Context, track *common.LearningTrack) (learn.Pools, error) {
	if f.err != nil {
		return nil, f.err
	}
	if track == nil {
		return f.pools, nil
	}
	for _, p := range f.pools {
		if p.Track == *track {
			return learn.Pools{p}, nil
		}
	}
	return learn.Pools{}, nil
}

func testSettings() learn.LearningSettings {
	return learn.LearningSettings{
		UserID: testUserID, DesiredRetention: 0.9, NewKanaPerDay: 15, NewKanjiPerDay: 10,
		MaxReviewsPerDay: 200, SessionSize: 20, LessonSize: 5, DailyGoalAnswers: 20,
		DayStartsAtHour: 4, AnswerMode: common.AnswerModeMixed, ShowRetention: true,
	}
}

// fakeRepo keeps the tables the progress services read in memory and
// implements every repo interface of the package.
type fakeRepo struct {
	settings    learn.LearningSettings
	tz          string
	cards       []learn.SRSCard
	stats       []learn.CardAnswerStats
	confusions  []learn.UserConfusion
	activity    []learn.DailyActivity
	goalDates   []time.Time
	answers     []learn.ReviewLog
	sessions    []learn.StudySession
	targetLevel string
	activeMs    int64
	introduced  map[common.SRSItemType]int
	reviewCount int
	trackOf     map[uuid.UUID]common.LearningTrack

	statsFilter *learn.AnswerStatsFilter
}

func newFakeRepo() *fakeRepo {
	trackOf := map[uuid.UUID]common.LearningTrack{}
	for _, p := range testPools() {
		for _, item := range p.Items {
			trackOf[item.ID] = p.Track
		}
	}
	return &fakeRepo{settings: testSettings(), introduced: map[common.SRSItemType]int{}, trackOf: trackOf}
}

// card adds a card of the item with the given facet, state and stability;
// mods change other fields.
func (f *fakeRepo) card(
	itemType common.SRSItemType, itemID uuid.UUID, facet common.CardFacet, state srs.State, stability float64, mods ...func(*learn.SRSCard),
) uuid.UUID {
	last := testNow.Add(-24 * time.Hour)
	c := learn.SRSCard{
		ID: uuid.New(), UserID: testUserID, ItemType: itemType, ItemID: itemID, Facet: facet, State: state,
		Due: testNow.Add(24 * time.Hour), Stability: stability, Difficulty: 5, Reps: 3, LastReviewAt: &last,
		IntroducedAt: testNow.Add(-7 * 24 * time.Hour),
	}
	for _, mod := range mods {
		mod(&c)
	}
	f.cards = append(f.cards, c)
	return c.ID
}

func (f *fakeRepo) EnsureSettings(context.Context, uuid.UUID) (learn.LearningSettings, error) {
	return f.settings, nil
}

func (f *fakeRepo) GetTimeZone(context.Context, uuid.UUID) (string, error) { return f.tz, nil }

func (f *fakeRepo) ListUserCards(context.Context, uuid.UUID) ([]learn.SRSCard, error) {
	return slices.Clone(f.cards), nil
}

func (f *fakeRepo) ListCardAnswerStats(_ context.Context, _ uuid.UUID, filter learn.AnswerStatsFilter) ([]learn.CardAnswerStats, error) {
	f.statsFilter = &filter
	if filter.CardIDs == nil {
		return slices.Clone(f.stats), nil
	}
	var out []learn.CardAnswerStats
	for _, s := range f.stats {
		if slices.Contains(filter.CardIDs, s.CardID) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListConfusions(context.Context, uuid.UUID) ([]learn.UserConfusion, error) {
	return f.confusions, nil
}

func (f *fakeRepo) ListGoalMetDates(context.Context, uuid.UUID) ([]time.Time, error) {
	return f.goalDates, nil
}

func (f *fakeRepo) GetDailyActivity(_ context.Context, _ uuid.UUID, localDate string) (learn.DailyActivity, error) {
	for _, a := range f.activity {
		if formatLocalDate(a.LocalDate) == localDate {
			return a, nil
		}
	}
	return learn.DailyActivity{}, nil
}

func (f *fakeRepo) ListDailyActivity(_ context.Context, _ uuid.UUID, from, to string) ([]learn.DailyActivity, error) {
	var out []learn.DailyActivity
	for _, a := range f.activity {
		if d := formatLocalDate(a.LocalDate); d >= from && d <= to {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeRepo) CountDueCards(_ context.Context, _ uuid.UUID, before time.Time, _ *common.LearningTrack) (int, error) {
	due, err := f.ListDueCards(context.Background(), testUserID, before, nil)
	return len(due), err
}

func (f *fakeRepo) ListDueCards(_ context.Context, _ uuid.UUID, before time.Time, track *common.LearningTrack) ([]learn.SRSCard, error) {
	var out []learn.SRSCard
	for _, c := range f.cards {
		if c.State == srs.StateNew || c.Due.After(before) || (track != nil && f.trackOf[c.ItemID] != *track) {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeRepo) CountItemsIntroducedSince(_ context.Context, _ uuid.UUID, itemType common.SRSItemType, _ time.Time) (int, error) {
	return f.introduced[itemType], nil
}

func (f *fakeRepo) CountReviewAnswersSince(context.Context, uuid.UUID, time.Time) (int, error) {
	return f.reviewCount, nil
}

func (f *fakeRepo) SumActiveMs(context.Context, uuid.UUID) (int64, error) { return f.activeMs, nil }

func (f *fakeRepo) GetTargetLevel(context.Context, uuid.UUID) (string, error) {
	return f.targetLevel, nil
}

func (f *fakeRepo) ListRecentSessions(_ context.Context, _ uuid.UUID, limit int) ([]learn.StudySession, error) {
	return f.sessions[:min(limit, len(f.sessions))], nil
}

func (f *fakeRepo) ListItemCards(_ context.Context, _ uuid.UUID, itemType common.SRSItemType, itemID uuid.UUID) ([]learn.SRSCard, error) {
	var out []learn.SRSCard
	for _, c := range f.cards {
		if c.ItemType == itemType && c.ItemID == itemID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListRecentAnswers(_ context.Context, _ uuid.UUID, cardIDs []uuid.UUID, limit int) ([]learn.ReviewLog, error) {
	var out []learn.ReviewLog
	for _, a := range f.answers {
		if slices.Contains(cardIDs, a.CardID) && len(out) < limit {
			out = append(out, a)
		}
	}
	return out, nil
}

// fakeKana serves the slugs of the test pools' kana.
type fakeKana struct{}

func (fakeKana) ListKana(_ context.Context, script common.KanaScript) ([]content.Kana, error) {
	if script == common.KanaScriptHiragana {
		return []content.Kana{{ID: hiraA, Slug: "a"}, {ID: hiraI, Slug: "i"}, {ID: hiraGa, Slug: "ga"}}, nil
	}
	return []content.Kana{{ID: kataA, Slug: "a"}}, nil
}

func date(s string) time.Time {
	d, err := parseLocalDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func stat(cardID uuid.UUID, qt common.QuestionType, answers, correct, reviews, passed, timed int, timedMs int64) learn.CardAnswerStats {
	return learn.CardAnswerStats{
		CardID: cardID, QuestionType: qt, Answers: answers, Correct: correct, Reviews: reviews,
		Passed: passed, TimedAnswers: timed, TimedMs: timedMs,
	}
}
