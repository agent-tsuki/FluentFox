package learn

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/srs"
)

var (
	testUserID = uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	otherUser  = uuid.MustParse("00000000-0000-0000-0000-0000000000bb")
	// 10:00 UTC; with day_starts_at_hour 4 the UTC study day began at 04:00.
	testNow = time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
)

func defaultSettings() LearningSettings {
	return LearningSettings{
		UserID: testUserID, DesiredRetention: 0.9, NewKanaPerDay: 15, NewKanjiPerDay: 10,
		MaxReviewsPerDay: 200, SessionSize: 20, LessonSize: 5, DailyGoalAnswers: 20,
		DayStartsAtHour: 4, AnswerMode: common.AnswerModeMixed, ShowRetention: true,
	}
}

// fakeRepo keeps every learning table in memory. It implements all the
// consumer interfaces; tx is ignored.
type fakeRepo struct {
	settings   LearningSettings
	tz         string
	cards      map[uuid.UUID]*SRSCard
	sessions   map[uuid.UUID]*StudySession
	logs       []ReviewLog
	confusions []UserConfusion
	activity   map[string]*DailyActivity
	samples    map[common.QuestionType][]BaselineSample

	// Counters the fake does not derive from cards.
	learned          int
	introducedToday  map[common.SRSItemType]int
	reviewAnswers    int
	err              error
	updatedSettings  *LearningSettings
	createdCards     []SRSCard
	completedSession *StudySession
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		settings:        defaultSettings(),
		cards:           map[uuid.UUID]*SRSCard{},
		sessions:        map[uuid.UUID]*StudySession{},
		activity:        map[string]*DailyActivity{},
		samples:         map[common.QuestionType][]BaselineSample{},
		introducedToday: map[common.SRSItemType]int{},
	}
}

func (f *fakeRepo) addCard(c SRSCard) *SRSCard {
	c.UserID = testUserID
	f.cards[c.ID] = &c
	return &c
}

func (f *fakeRepo) addSession(kind common.SessionKind, track *common.LearningTrack) *StudySession {
	s := &StudySession{ID: uuid.New(), UserID: testUserID, Kind: kind, Track: track, StartedAt: testNow.Add(-time.Hour)}
	f.sessions[s.ID] = s
	return s
}

func (f *fakeRepo) Transaction(_ context.Context, fn func(*gorm.DB) error) error {
	return fn(nil)
}

func (f *fakeRepo) EnsureSettings(context.Context, uuid.UUID) (LearningSettings, error) {
	return f.settings, f.err
}

func (f *fakeRepo) UpdateSettings(_ context.Context, s *LearningSettings) error {
	copied := *s
	f.updatedSettings = &copied
	return f.err
}

func (f *fakeRepo) GetTimeZone(context.Context, uuid.UUID) (string, error) {
	return f.tz, nil
}

func (f *fakeRepo) CountDueCards(_ context.Context, _ uuid.UUID, before time.Time, _ *common.LearningTrack) (int, error) {
	return f.countDue(before), nil
}

func (f *fakeRepo) CountDueCardsTx(_ context.Context, _ *gorm.DB, _ uuid.UUID, before time.Time) (int, error) {
	return f.countDue(before), nil
}

func (f *fakeRepo) countDue(before time.Time) int {
	n := 0
	for _, c := range f.cards {
		if c.State != srs.StateNew && !c.Due.After(before) {
			n++
		}
	}
	return n
}

func (f *fakeRepo) CountLearnedItems(context.Context, uuid.UUID, *common.LearningTrack) (int, error) {
	return f.learned, nil
}

func (f *fakeRepo) CountItemsIntroducedSince(_ context.Context, _ uuid.UUID, itemType common.SRSItemType, _ time.Time) (int, error) {
	return f.introducedToday[itemType], nil
}

func (f *fakeRepo) ListCardKeys(_ context.Context, _ uuid.UUID, itemType common.SRSItemType) ([]CardKey, error) {
	var keys []CardKey
	for _, c := range f.cards {
		if c.ItemType == itemType {
			keys = append(keys, CardKey{ItemID: c.ItemID, Facet: c.Facet})
		}
	}
	return keys, nil
}

func (f *fakeRepo) GetDailyActivity(_ context.Context, _ uuid.UUID, localDate string) (DailyActivity, error) {
	if a, ok := f.activity[localDate]; ok {
		return *a, nil
	}
	return DailyActivity{}, nil
}

func (f *fakeRepo) CountReviewAnswersSince(context.Context, uuid.UUID, time.Time) (int, error) {
	return f.reviewAnswers, nil
}

func (f *fakeRepo) ListConfusions(context.Context, uuid.UUID) ([]UserConfusion, error) {
	return f.confusions, nil
}

func (f *fakeRepo) CreateSession(_ context.Context, s *StudySession) error {
	copied := *s
	f.sessions[s.ID] = &copied
	return f.err
}

func (f *fakeRepo) GetSession(_ context.Context, userID, sessionID uuid.UUID) (StudySession, error) {
	s, ok := f.sessions[sessionID]
	if !ok || s.UserID != userID {
		return StudySession{}, gorm.ErrRecordNotFound
	}
	return *s, nil
}

func (f *fakeRepo) LockSession(_ context.Context, _ *gorm.DB, sessionID uuid.UUID) (StudySession, error) {
	s, ok := f.sessions[sessionID]
	if !ok {
		return StudySession{}, gorm.ErrRecordNotFound
	}
	return *s, nil
}

func (f *fakeRepo) ListDueCards(_ context.Context, _ uuid.UUID, now time.Time, _ *common.LearningTrack) ([]SRSCard, error) {
	var out []SRSCard
	for _, c := range f.cards {
		if c.State != srs.StateNew && !c.Due.After(now) {
			out = append(out, *c)
		}
	}
	slices.SortFunc(out, func(a, b SRSCard) int { return a.Due.Compare(b.Due) })
	return out, nil
}

func (f *fakeRepo) ListCardsByIDs(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]SRSCard, error) {
	var out []SRSCard
	for _, id := range ids {
		if c, ok := f.cards[id]; ok {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListBaselineSamples(_ context.Context, _ uuid.UUID, qt common.QuestionType, _ int) ([]BaselineSample, error) {
	return f.samples[qt], nil
}

func (f *fakeRepo) ListItemChars(_ context.Context, _ common.SRSItemType, ids []uuid.UUID) ([]ItemChar, error) {
	var out []ItemChar
	for _, id := range ids {
		if char, ok := testChars[id]; ok {
			out = append(out, ItemChar{ID: id, Char: char})
		}
	}
	return out, nil
}

func (f *fakeRepo) FindReviewLogs(_ context.Context, _ *gorm.DB, userID uuid.UUID, eventIDs []uuid.UUID) ([]ReviewLog, error) {
	var out []ReviewLog
	for _, l := range f.logs {
		if l.UserID == userID && slices.Contains(eventIDs, l.ClientEventID) {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListExistingCardIDs(_ context.Context, _ *gorm.DB, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for _, id := range ids {
		if c, ok := f.cards[id]; ok && c.UserID == userID {
			out = append(out, id)
		}
	}
	return out, nil
}

func (f *fakeRepo) CreateCards(_ context.Context, _ *gorm.DB, cards []SRSCard) error {
	for _, c := range cards {
		f.createdCards = append(f.createdCards, c)
		copied := c
		f.cards[c.ID] = &copied
	}
	return nil
}

func (f *fakeRepo) LockCards(_ context.Context, _ *gorm.DB, userID uuid.UUID, ids []uuid.UUID) ([]SRSCard, error) {
	var out []SRSCard
	for _, id := range ids {
		if c, ok := f.cards[id]; ok && c.UserID == userID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListCardsRatedSince(_ context.Context, _ *gorm.DB, cardIDs []uuid.UUID, since time.Time) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for _, l := range f.logs {
		if slices.Contains(cardIDs, l.CardID) && !l.AnsweredAt.Before(since) && l.Rating != nil {
			out = append(out, l.CardID)
		}
	}
	return out, nil
}

func (f *fakeRepo) SaveCard(_ context.Context, _ *gorm.DB, card *SRSCard) error {
	copied := *card
	f.cards[card.ID] = &copied
	return nil
}

func (f *fakeRepo) InsertReviewLog(_ context.Context, _ *gorm.DB, log *ReviewLog) error {
	f.logs = append(f.logs, *log)
	return nil
}

func (f *fakeRepo) UpsertConfusion(_ context.Context, _ *gorm.DB, c *UserConfusion) error {
	for i := range f.confusions {
		if f.confusions[i].TargetItemID == c.TargetItemID && f.confusions[i].ChosenItemID == c.ChosenItemID {
			f.confusions[i].Times++
			return nil
		}
	}
	copied := *c
	copied.Times = 1
	f.confusions = append(f.confusions, copied)
	return nil
}

func (f *fakeRepo) AddDailyActivity(_ context.Context, _ *gorm.DB, delta *DailyActivity, localDate string) (DailyActivity, error) {
	day, ok := f.activity[localDate]
	if !ok {
		day = &DailyActivity{UserID: delta.UserID, DueAtDayStart: delta.DueAtDayStart}
		f.activity[localDate] = day
	}
	day.ReviewsDone += delta.ReviewsDone
	day.NewLearned += delta.NewLearned
	day.CorrectCount += delta.CorrectCount
	day.AgainCount += delta.AgainCount
	day.ActiveMs += delta.ActiveMs
	return *day, nil
}

func (f *fakeRepo) MarkGoalMet(_ context.Context, _ *gorm.DB, _ uuid.UUID, localDate string) error {
	f.activity[localDate].GoalMet = true
	return nil
}

func (f *fakeRepo) ListSessionLogs(_ context.Context, _ *gorm.DB, sessionID uuid.UUID) ([]ReviewLog, error) {
	var out []ReviewLog
	for _, l := range f.logs {
		if l.SessionID == sessionID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeRepo) CompleteSession(_ context.Context, _ *gorm.DB, s *StudySession) error {
	copied := *s
	f.completedSession = &copied
	f.sessions[s.ID] = &copied
	return nil
}

// Test content: two hiragana units, one katakana unit and two kanji units.
var (
	unitA       = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	unitZa      = uuid.MustParse("10000000-0000-0000-0000-000000000002")
	unitKA      = uuid.MustParse("10000000-0000-0000-0000-000000000003")
	unitK1      = uuid.MustParse("10000000-0000-0000-0000-000000000004")
	unitK2      = uuid.MustParse("10000000-0000-0000-0000-000000000005")
	kanaA       = uuid.MustParse("20000000-0000-0000-0000-000000000001")
	kanaI       = uuid.MustParse("20000000-0000-0000-0000-000000000002")
	kanaO       = uuid.MustParse("20000000-0000-0000-0000-000000000003")
	kanaJi      = uuid.MustParse("20000000-0000-0000-0000-000000000004")
	kanaDji     = uuid.MustParse("20000000-0000-0000-0000-000000000005")
	kanaZu      = uuid.MustParse("20000000-0000-0000-0000-000000000006")
	kanaKA      = uuid.MustParse("20000000-0000-0000-0000-000000000007")
	kanjiOne    = uuid.MustParse("30000000-0000-0000-0000-000000000001")
	kanjiTwo    = uuid.MustParse("30000000-0000-0000-0000-000000000002")
	kanjiMiddle = uuid.MustParse("30000000-0000-0000-0000-000000000003")
	kanjiHalf   = uuid.MustParse("30000000-0000-0000-0000-000000000004")
	kanjiDay    = uuid.MustParse("30000000-0000-0000-0000-000000000005")
)

var testChars = map[uuid.UUID]string{
	kanaA: "あ", kanaI: "い", kanaO: "お", kanaJi: "じ", kanaDji: "ぢ", kanaZu: "ず", kanaKA: "ア",
	kanjiOne: "一", kanjiTwo: "二", kanjiMiddle: "中", kanjiHalf: "半", kanjiDay: "日",
}

type fakeContent struct {
	err error
}

func (f *fakeContent) ListKanaUnits(_ context.Context, script common.KanaScript) ([]content.KanaUnit, error) {
	if script == common.KanaScriptKatakana {
		return []content.KanaUnit{{ID: unitKA, Script: script, Slug: "a", Title: "ア row", Position: 1}}, f.err
	}
	return []content.KanaUnit{
		// Listed out of order: the pool must sort by position.
		{ID: unitZa, Script: script, Slug: "za", Title: "ざ row", Position: 2},
		{ID: unitA, Script: script, Slug: "a", Title: "あ row", Position: 1},
	}, f.err
}

func (f *fakeContent) ListKana(_ context.Context, script common.KanaScript) ([]content.Kana, error) {
	if script == common.KanaScriptKatakana {
		return []content.Kana{{ID: kanaKA, Script: script, Slug: "a", Character: "ア", Romaji: "a", UnitID: unitKA, Position: 1}}, nil
	}
	return []content.Kana{
		{ID: kanaJi, Slug: "ji", Character: "じ", Romaji: "ji", AltRomaji: database.TextArray{"zi"}, UnitID: unitZa, Position: 4},
		{ID: kanaDji, Slug: "dji", Character: "ぢ", Romaji: "dji", AltRomaji: database.TextArray{"di", "ji"}, UnitID: unitZa, Position: 5},
		{ID: kanaZu, Slug: "zu", Character: "ず", Romaji: "zu", UnitID: unitZa, Position: 6},
		{ID: kanaA, Slug: "a", Character: "あ", Romaji: "a", UnitID: unitA, Position: 1},
		{ID: kanaI, Slug: "i", Character: "い", Romaji: "i", UnitID: unitA, Position: 2},
		{ID: kanaO, Slug: "o", Character: "お", Romaji: "o", UnitID: unitA, Position: 3},
	}, nil
}

func (f *fakeContent) ListKanaLookAlikes(_ context.Context, script common.KanaScript) ([]content.KanaLookAlikeRow, error) {
	if script == common.KanaScriptKatakana {
		return nil, nil
	}
	return []content.KanaLookAlikeRow{
		{KanaID: kanaA, OtherSlug: "o", OtherChar: "お"},
		{KanaID: kanaO, OtherSlug: "a", OtherChar: "あ"},
	}, nil
}

func (f *fakeContent) ListKanjiUnits(_ context.Context, level common.JLPTLevel) ([]content.KanjiUnit, error) {
	return []content.KanjiUnit{
		{ID: unitK1, Level: level, Slug: "u01", Title: "Numbers", Position: 1},
		{ID: unitK2, Level: level, Slug: "u02", Title: "Time", Position: 2},
	}, f.err
}

func (f *fakeContent) ListKanji(_ context.Context, level common.JLPTLevel) ([]content.Kanji, error) {
	return []content.Kanji{
		{ID: kanjiOne, Char: "一", Level: level, UnitID: unitK1, Position: 1, Meanings: database.TextArray{"one"},
			Onyomi: database.TextArray{"イチ", "イツ"}, KeyReading: "イチ"},
		{ID: kanjiTwo, Char: "二", Level: level, UnitID: unitK1, Position: 2, Meanings: database.TextArray{"two"},
			Onyomi: database.TextArray{"ニ"}, KeyReading: "ニ"},
		{ID: kanjiMiddle, Char: "中", Level: level, UnitID: unitK1, Position: 3, Meanings: database.TextArray{"middle", "inside"},
			Onyomi: database.TextArray{"チュウ"}, KeyReading: "チュウ"},
		{ID: kanjiHalf, Char: "半", Level: level, UnitID: unitK1, Position: 4, Meanings: database.TextArray{"half", "middle"},
			Onyomi: database.TextArray{"ハン"}, KeyReading: "ハン"},
		{ID: kanjiDay, Char: "日", Level: level, UnitID: unitK2, Position: 1, Meanings: database.TextArray{"day", "sun"},
			Synonyms: database.TextArray{"daytime"}, Onyomi: database.TextArray{"ニチ", "ジツ"}, KeyReading: "ニチ"},
	}, nil
}

func (f *fakeContent) ListKanjiKunReadings(context.Context, []uuid.UUID) ([]content.KanjiKunReading, error) {
	tsu := "つ"
	return []content.KanjiKunReading{
		{KanjiID: kanjiOne, Kana: "ひと", Okurigana: &tsu},
		{KanjiID: kanjiDay, Kana: "ひ"},
	}, nil
}

func (f *fakeContent) ListKanjiLookAlikes(context.Context, []uuid.UUID) ([]content.KanjiLookAlikeRow, error) {
	return []content.KanjiLookAlikeRow{{KanjiID: kanjiOne, OtherChar: "二"}}, nil
}

// fixedRand returns the queued values from IntN (then 1) and never shuffles,
// so questions are deterministic. 1 is never the "variety" pick.
type fixedRand struct {
	ints []int
}

func (r *fixedRand) IntN(n int) int {
	if len(r.ints) == 0 {
		return 1 % n
	}
	v := r.ints[0]
	r.ints = r.ints[1:]
	return v % n
}

func (r *fixedRand) Shuffle(int, func(i, j int)) {}

func loadTestPool(track common.LearningTrack) *Pool {
	pool, err := NewCatalog(&fakeContent{}).Load(context.Background(), track)
	if err != nil {
		panic(err)
	}
	return pool
}

func ptr[T any](v T) *T { return &v }

var errBoom = errors.New("boom")
