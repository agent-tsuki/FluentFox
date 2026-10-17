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

func TestStartTestService(t *testing.T) {
	tests := []struct {
		name      string
		req       StartTestRequest
		setup     func(r *fakeRepo)
		wantItems []uuid.UUID
		wantKind  common.SessionKind
		wantScope string
		wantErr   error
	}{
		{
			name: "a unit quiz asks every item of the unit", req: StartTestRequest{Kind: "unit_quiz", Track: "hiragana", Scope: "a"},
			wantItems: []uuid.UUID{kanaA, kanaI, kanaO}, wantKind: common.SessionKindUnitQuiz, wantScope: "a",
		},
		{
			name: "a kanji unit quiz asks both facets", req: StartTestRequest{Kind: "unit_quiz", Track: "kanji_n5", Scope: "u02"},
			wantItems: []uuid.UUID{kanjiDay, kanjiDay}, wantKind: common.SessionKindUnitQuiz, wantScope: "u02",
		},
		{
			name: "a skip test covers a group", req: StartTestRequest{Kind: "skip", Track: "hiragana", Scope: "dakuten"},
			wantItems: []uuid.UUID{kanaJi, kanaDji, kanaZu}, wantKind: common.SessionKindSkipTest, wantScope: "dakuten",
		},
		{
			name: "a kanji skip test covers a kanji group", req: StartTestRequest{Kind: "skip", Track: "kanji_n5", Scope: "nature"},
			wantItems: []uuid.UUID{kanjiDay, kanjiDay}, wantKind: common.SessionKindSkipTest, wantScope: "nature",
		},
		{
			name: "a skip test without a scope covers the track", req: StartTestRequest{Kind: "skip", Track: "hiragana"},
			wantItems: []uuid.UUID{kanaA, kanaI, kanaO, kanaJi, kanaDji, kanaZu}, wantKind: common.SessionKindSkipTest, wantScope: "all",
		},
		{
			name: "a final test covers the track whatever the scope", req: StartTestRequest{Kind: "final", Track: "katakana", Scope: "a"},
			wantItems: []uuid.UUID{kanaKA}, wantKind: common.SessionKindFinalTest, wantScope: "all",
		},
		{
			name: "a learned item is asked with its own card",
			req:  StartTestRequest{Kind: "unit_quiz", Track: "hiragana", Scope: "a"},
			setup: func(r *fakeRepo) {
				c := learnedKana(kanaA, 10, 1)
				c.ID = uuid.New()
				r.addCard(c)
			},
			wantItems: []uuid.UUID{kanaA, kanaI, kanaO}, wantKind: common.SessionKindUnitQuiz, wantScope: "a",
		},
		{
			name:    "an unknown unit is not found",
			req:     StartTestRequest{Kind: "unit_quiz", Track: "hiragana", Scope: "zz"},
			wantErr: exceptions.ErrUnitNotFound,
		},
		{
			name:    "an unknown group is rejected",
			req:     StartTestRequest{Kind: "skip", Track: "hiragana", Scope: "foreign"},
			wantErr: exceptions.ErrTestScopeInvalid,
		},
		{
			name:    "an unknown kind is rejected",
			req:     StartTestRequest{Kind: "midterm", Track: "hiragana"},
			wantErr: exceptions.ErrTestKindInvalid,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tc.setup != nil {
				tc.setup(repo)
			}
			svc := NewStartTestService(repo, NewCatalog(&fakeContent{}), NewQuestionBuilder(&fixedRand{}), &fixedRand{}, zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Start(context.Background(), testUserID, &tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if len(got.Questions) != len(tc.wantItems) {
				t.Fatalf("questions = %d, want %d", len(got.Questions), len(tc.wantItems))
			}
			for i, q := range got.Questions {
				if q.ItemID != tc.wantItems[i] {
					t.Errorf("question %d is for %s, want %s", i, testChars[q.ItemID], testChars[tc.wantItems[i]])
				}
				card := repo.cardFor(q.ItemID, common.CardFacet(q.Facet))
				if card != nil && q.CardID != card.ID {
					t.Errorf("question %d card id = %s, want the user's card %s", i, q.CardID, card.ID)
				}
			}
			stored := repo.sessions[got.Session.ID]
			if stored == nil || stored.Kind != tc.wantKind || stored.Scope == nil || *stored.Scope != tc.wantScope {
				t.Errorf("stored session = %+v, want %s with scope %q", stored, tc.wantKind, tc.wantScope)
			}
		})
	}
}

func (f *fakeRepo) cardFor(item uuid.UUID, facet common.CardFacet) *SRSCard {
	for _, c := range f.cards {
		if c.ItemID == item && c.Facet == facet {
			return c
		}
	}
	return nil
}

// testSession adds a test session over scope with one answer per item: the
// items in right are answered correctly, the others wrongly.
func testSession(r *fakeRepo, kind common.SessionKind, track common.LearningTrack, scope string, right, wrong []uuid.UUID) uuid.UUID {
	s := r.addSession(kind, &track)
	s.Scope = &scope
	itemType := common.LEARNING_TRACK_ITEM_TYPE[track]
	add := func(items []uuid.UUID, correct bool) {
		for _, item := range items {
			for _, facet := range common.SRS_ITEM_FACETS[itemType] {
				c := r.addCard(newCard(itemType, item, facet))
				r.logs = append(r.logs, sessionLog(s, c, nil, correct, 1500, time.Minute))
			}
		}
	}
	add(right, true)
	add(wrong, false)
	return s.ID
}

func TestCompleteTestSession(t *testing.T) {
	tests := []struct {
		name  string
		setup func(r *fakeRepo) uuid.UUID
		want  *TestScore
		check func(t *testing.T, r *fakeRepo)
	}{
		{
			name: "a passed unit quiz keeps the best score and completes the unit",
			setup: func(r *fakeRepo) uuid.UUID {
				return testSession(r, common.SessionKindUnitQuiz, common.LearningTrackHiragana, "a", []uuid.UUID{kanaA, kanaI, kanaO}, nil)
			},
			want: &TestScore{Kind: "unit_quiz", Score: 3, Total: 3, Passed: true, PassMark: common.KANA_UNIT_QUIZ_PASS_MARK},
			check: func(t *testing.T, r *fakeRepo) {
				if len(r.unitQuizzes) != 1 || r.unitQuizzes[0].UnitID != unitA || *r.unitQuizzes[0].QuizBestPct != 100 || r.unitQuizzes[0].CompletedAt == nil {
					t.Errorf("unit quizzes = %+v, want あ row at 100 %% and completed", r.unitQuizzes)
				}
			},
		},
		{
			name: "a failed unit quiz records its score without completing the unit",
			setup: func(r *fakeRepo) uuid.UUID {
				return testSession(r, common.SessionKindUnitQuiz, common.LearningTrackHiragana, "a", []uuid.UUID{kanaA, kanaI}, []uuid.UUID{kanaO})
			},
			want: &TestScore{Kind: "unit_quiz", Score: 2, Total: 3, Passed: false, PassMark: common.KANA_UNIT_QUIZ_PASS_MARK},
			check: func(t *testing.T, r *fakeRepo) {
				if len(r.unitQuizzes) != 1 || *r.unitQuizzes[0].QuizBestPct != 66 || r.unitQuizzes[0].CompletedAt != nil {
					t.Errorf("unit quizzes = %+v, want 66 %% and not completed", r.unitQuizzes)
				}
			},
		},
		{
			name: "a kanji unit quiz needs the stricter kanji pass mark",
			setup: func(r *fakeRepo) uuid.UUID {
				return testSession(r, common.SessionKindUnitQuiz, common.LearningTrackKanjiN5, "u01",
					[]uuid.UUID{kanjiOne, kanjiTwo, kanjiMiddle}, []uuid.UUID{kanjiHalf})
			},
			want: &TestScore{Kind: "unit_quiz", Score: 6, Total: 8, Passed: false, PassMark: common.KANJI_UNIT_QUIZ_PASS_MARK},
		},
		{
			name: "a passed skip test makes unlearned items known and leaves learned ones alone",
			setup: func(r *fakeRepo) uuid.UUID {
				id := testSession(r, common.SessionKindSkipTest, common.LearningTrackHiragana, "dakuten", []uuid.UUID{kanaJi, kanaDji, kanaZu}, nil)
				learning := newCard(common.SRSItemKana, kanaJi, common.CardFacetRecognition)
				learning.State, learning.Due = srs.StateLearning, testNow.Add(time.Minute)
				r.addCard(learning)
				return id
			},
			want: &TestScore{Kind: "skip", Score: 3, Total: 3, Passed: true, PassMark: common.SKIP_TEST_PASS_MARK},
			check: func(t *testing.T, r *fakeRepo) {
				if c := r.cardFor(kanaJi, common.CardFacetRecognition); c.State != srs.StateLearning {
					t.Errorf("じ card = %+v, want it still learning", c)
				}
				for _, item := range []uuid.UUID{kanaDji, kanaZu} {
					c := r.cardFor(item, common.CardFacetRecognition)
					if c.State != srs.StateReview || c.Due.Before(testNow.Add(srs.KnownCardMinDueAfter)) || c.Due.After(testNow.Add(srs.KnownCardMaxDueAfter)) {
						t.Errorf("%s card = %+v, want a known review card due in 1 to 7 days", testChars[item], c)
					}
				}
				if r.cardFor(kanaA, common.CardFacetRecognition) != nil {
					t.Error("a skip test made an item outside its scope known")
				}
			},
		},
		{
			name: "unanswered questions count against a skip test",
			setup: func(r *fakeRepo) uuid.UUID {
				return testSession(r, common.SessionKindSkipTest, common.LearningTrackHiragana, "all", []uuid.UUID{kanaA, kanaI, kanaO}, nil)
			},
			want: &TestScore{Kind: "skip", Score: 3, Total: 6, Passed: false, PassMark: common.SKIP_TEST_PASS_MARK},
			check: func(t *testing.T, r *fakeRepo) {
				for _, c := range r.cards {
					if c.State != srs.StateNew {
						t.Errorf("a failed skip test scheduled %s", testChars[c.ItemID])
					}
				}
			},
		},
		{
			name: "a final test scores the whole track",
			setup: func(r *fakeRepo) uuid.UUID {
				return testSession(r, common.SessionKindFinalTest, common.LearningTrackKatakana, "all", []uuid.UUID{kanaKA}, nil)
			},
			want: &TestScore{Kind: "final", Score: 1, Total: 1, Passed: true, PassMark: common.FINAL_TEST_PASS_MARK},
		},
		{
			name: "completing a test twice returns the stored result",
			setup: func(r *fakeRepo) uuid.UUID {
				id := testSession(r, common.SessionKindFinalTest, common.LearningTrackKatakana, "all", nil, nil)
				r.sessions[id].CompletedAt = ptr(testNow.Add(-time.Minute))
				r.testResults = []TestResult{{SessionID: id, Kind: common.TestKindFinal, Track: common.LearningTrackKatakana, Score: 1, Total: 1, Passed: true}}
				return id
			},
			want: &TestScore{Kind: "final", Score: 1, Total: 1, Passed: true, PassMark: common.FINAL_TEST_PASS_MARK},
			check: func(t *testing.T, r *fakeRepo) {
				if len(r.testResults) != 1 {
					t.Errorf("test results = %d, want the stored one only", len(r.testResults))
				}
			},
		},
		{
			name: "a practice session has no test score",
			setup: func(r *fakeRepo) uuid.UUID {
				return testSession(r, common.SessionKindPractice, common.LearningTrackHiragana, "weak", []uuid.UUID{kanaA}, nil)
			},
			check: func(t *testing.T, r *fakeRepo) {
				if len(r.testResults) != 0 {
					t.Error("a practice stored a test result")
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			id := tc.setup(repo)
			scores := NewTestScoreService(repo, NewCatalog(&fakeContent{}), nil, zap.NewNop())
			scores.now = func() time.Time { return testNow }
			svc := NewCompleteSessionService(repo, scores, zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Complete(context.Background(), testUserID, id)
			if err != nil {
				t.Fatalf("complete: %v", err)
			}
			if (got.Test == nil) != (tc.want == nil) || (got.Test != nil && *got.Test != *tc.want) {
				t.Errorf("test = %+v, want %+v", got.Test, tc.want)
			}
			if tc.want != nil && len(repo.testResults) != 1 {
				t.Errorf("test results = %+v, want one", repo.testResults)
			}
			if tc.check != nil {
				tc.check(t, repo)
			}
		})
	}
}
