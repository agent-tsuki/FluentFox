package learn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

// learnedCard is a review card last seen daysAgo with the given stability.
func learnedCard(itemType common.SRSItemType, item uuid.UUID, facet common.CardFacet, stability float64, daysAgo int) SRSCard {
	last := testNow.Add(-time.Duration(daysAgo) * 24 * time.Hour)
	return SRSCard{
		ID: cardID(testUserID, itemType, item, facet), ItemType: itemType, ItemID: item, Facet: facet,
		State: srs.StateReview, Stability: stability, Difficulty: 5, Reps: 4, ScheduledDays: daysAgo,
		Due: testNow.Add(24 * time.Hour), LastReviewAt: &last,
	}
}

func learnedKana(item uuid.UUID, stability float64, daysAgo int) SRSCard {
	return learnedCard(common.SRSItemKana, item, common.CardFacetRecognition, stability, daysAgo)
}

func newCard(itemType common.SRSItemType, item uuid.UUID, facet common.CardFacet) SRSCard {
	return SRSCard{ID: cardID(testUserID, itemType, item, facet), ItemType: itemType, ItemID: item, Facet: facet, State: srs.StateNew}
}

type questionWant struct {
	item uuid.UUID
	qt   common.QuestionType
}

func TestStartPracticeService(t *testing.T) {
	tests := []struct {
		name      string
		req       StartPracticeRequest
		setup     func(r *fakeRepo)
		want      []questionWant
		wantScope string
		wantErr   error
	}{
		{
			name: "weak asks the lowest recall chance and highest lapse rate first, without new cards",
			req:  StartPracticeRequest{Track: "hiragana", Mode: "weak"},
			setup: func(r *fakeRepo) {
				r.settings.SessionSize = 2
				r.addCard(learnedKana(kanaA, 30, 1))
				r.addCard(learnedKana(kanaI, 0.5, 30))
				lapsed := learnedKana(kanaO, 30, 1)
				lapsed.Lapses = 4
				r.addCard(lapsed)
				r.addCard(newCard(common.SRSItemKana, kanaJi, common.CardFacetRecognition))
			},
			want:      []questionWant{{item: kanaO}, {item: kanaI}},
			wantScope: "weak",
		},
		{
			name: "weak with nothing learned gives an empty session",
			req:  StartPracticeRequest{Track: "katakana", Mode: "weak"}, wantScope: "weak",
		},
		{
			name: "unit asks every item of the unit, learned or not",
			req:  StartPracticeRequest{Track: "hiragana", Mode: "unit", Unit: "za"},
			want: []questionWant{{item: kanaJi}, {item: kanaDji}, {item: kanaZu}}, wantScope: "unit:za",
		},
		{
			name: "unit asks both facets of a kanji",
			req:  StartPracticeRequest{Track: "kanji_n5", Mode: "unit", Unit: "u02"},
			want: []questionWant{{item: kanjiDay}, {item: kanjiDay}}, wantScope: "unit:u02",
		},
		{
			name:    "unit practice needs a unit",
			req:     StartPracticeRequest{Track: "hiragana", Mode: "unit"},
			wantErr: exceptions.ErrPracticeUnitRequired,
		},
		{
			name:    "an unknown unit is not found",
			req:     StartPracticeRequest{Track: "hiragana", Mode: "unit", Unit: "zz"},
			wantErr: exceptions.ErrUnitNotFound,
		},
		{
			name: "lookalike drills the given pair",
			req:  StartPracticeRequest{Track: "hiragana", Mode: "lookalike", Pair: "あ:お"},
			want: []questionWant{
				{item: kanaA, qt: common.QuestionKanaLookalike}, {item: kanaO, qt: common.QuestionKanaLookalike},
			},
			wantScope: "lookalike:あ:お",
		},
		{
			name: "kanji look-alikes are told apart by meaning",
			req:  StartPracticeRequest{Track: "kanji_n5", Mode: "lookalike", Pair: "一:二"},
			want: []questionWant{
				{item: kanjiOne, qt: common.QuestionKanjiLookalike}, {item: kanjiTwo, qt: common.QuestionKanjiLookalike},
			},
			wantScope: "lookalike:一:二",
		},
		{
			name: "lookalike without a pair takes the user's mix-ups, then look-alikes of learned items",
			req:  StartPracticeRequest{Track: "hiragana", Mode: "lookalike"},
			setup: func(r *fakeRepo) {
				r.confusions = []UserConfusion{
					{ItemType: common.SRSItemKana, TargetItemID: kanaJi, ChosenItemID: kanaZu, Times: 3},
					{ItemType: common.SRSItemKana, TargetItemID: kanaKA, ChosenItemID: kanaA, Times: 9},
				}
				r.addCard(learnedKana(kanaA, 10, 1))
			},
			want: []questionWant{
				{item: kanaJi, qt: common.QuestionKanaLookalike}, {item: kanaZu, qt: common.QuestionKanaLookalike},
				{item: kanaA, qt: common.QuestionKanaLookalike}, {item: kanaO, qt: common.QuestionKanaLookalike},
			},
			wantScope: "lookalike",
		},
		{
			name:    "a pair needs two characters",
			req:     StartPracticeRequest{Track: "hiragana", Mode: "lookalike", Pair: "あ"},
			wantErr: exceptions.ErrPracticePairInvalid,
		},
		{
			name:    "a pair must be of the track",
			req:     StartPracticeRequest{Track: "hiragana", Mode: "lookalike", Pair: "あ:ア"},
			wantErr: exceptions.ErrPracticePairInvalid,
		},
		{
			name: "speed asks learned kana as speed questions",
			req:  StartPracticeRequest{Track: "hiragana", Mode: "speed"},
			setup: func(r *fakeRepo) {
				r.addCard(learnedKana(kanaA, 10, 1))
				r.addCard(newCard(common.SRSItemKana, kanaI, common.CardFacetRecognition))
			},
			want:      []questionWant{{item: kanaA, qt: common.QuestionKanaSpeed}},
			wantScope: "speed",
		},
		{
			name: "match asks kanji as plain multiple choice even when typing is allowed",
			req:  StartPracticeRequest{Track: "kanji_n5", Mode: "match"},
			setup: func(r *fakeRepo) {
				r.settings.AnswerMode = common.AnswerModeTyped
				r.addCard(learnedCard(common.SRSItemKanji, kanjiOne, common.CardFacetMeaning, 10, 1))
			},
			want:      []questionWant{{item: kanjiOne, qt: common.QuestionKanjiMeaningMCQ}},
			wantScope: "match",
		},
		{
			name:    "an unknown mode is rejected",
			req:     StartPracticeRequest{Track: "hiragana", Mode: "sprint"},
			wantErr: exceptions.ErrPracticeModeInvalid,
		},
		{
			name:    "practice needs a track",
			req:     StartPracticeRequest{Mode: "weak"},
			wantErr: exceptions.ErrLearningTrackInvalid,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tc.setup != nil {
				tc.setup(repo)
			}
			svc := NewStartPracticeService(repo, NewCatalog(&fakeContent{}), NewQuestionBuilder(&fixedRand{}), &fixedRand{}, zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Start(context.Background(), testUserID, &tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if len(repo.sessions) != 0 {
					t.Error("a rejected practice stored a session")
				}
				return
			}
			checkQuestions(t, got.Questions, tc.want)
			stored := repo.sessions[got.Session.ID]
			if stored == nil || stored.Kind != common.SessionKindPractice || stored.Scope == nil || *stored.Scope != tc.wantScope {
				t.Errorf("stored session = %+v, want a practice with scope %q", stored, tc.wantScope)
			}
		})
	}
}

// checkQuestions compares items and, where want names one, question types.
// Every question must carry the card id its answer will use.
func checkQuestions(t *testing.T, got []Question, want []questionWant) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("questions = %d (%+v), want %d", len(got), got, len(want))
	}
	for i, q := range got {
		if q.ItemID != want[i].item {
			t.Errorf("question %d is for %s, want %s", i, testChars[q.ItemID], testChars[want[i].item])
		}
		if want[i].qt != "" && q.QuestionType != string(want[i].qt) {
			t.Errorf("question %d type = %s, want %s", i, q.QuestionType, want[i].qt)
		}
		if q.CardID != cardID(testUserID, common.SRSItemType(q.ItemType), q.ItemID, common.CardFacet(q.Facet)) {
			t.Errorf("question %d card id = %s, want the derived id", i, q.CardID)
		}
	}
}
