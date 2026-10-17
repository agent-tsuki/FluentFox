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

func TestStartLessonService(t *testing.T) {
	tests := []struct {
		name          string
		track         string
		setup         func(r *fakeRepo)
		wantItems     []string
		wantQuestions int
		wantErr       error
	}{
		{name: "takes the first items in path order up to the lesson size", track: "hiragana",
			wantItems: []string{"あ", "い", "お", "じ", "ぢ"}, wantQuestions: 5},
		{name: "skips items that already have their card", track: "hiragana",
			setup: func(r *fakeRepo) {
				r.addCard(SRSCard{ID: uuid.New(), ItemType: common.SRSItemKana, ItemID: kanaA, Facet: common.CardFacetRecognition, State: srs.StateLearning})
			},
			wantItems: []string{"い", "お", "じ", "ぢ", "ず"}, wantQuestions: 5},
		{name: "stops at the daily new limit", track: "hiragana",
			setup:     func(r *fakeRepo) { r.introducedToday[common.SRSItemKana] = 13 },
			wantItems: []string{"あ", "い"}, wantQuestions: 2},
		{name: "a used-up daily limit gives an empty lesson", track: "katakana",
			setup:     func(r *fakeRepo) { r.settings.NewKanaPerDay = 0 },
			wantItems: []string{}, wantQuestions: 0},
		{name: "each kanji asks its meaning and its reading", track: "kanji_n5",
			setup:     func(r *fakeRepo) { r.settings.LessonSize = 2 },
			wantItems: []string{"一", "二"}, wantQuestions: 4},
		{name: "a kanji missing only its reading card asks only the reading", track: "kanji_n5",
			setup: func(r *fakeRepo) {
				r.settings.LessonSize = 1
				r.addCard(SRSCard{ID: uuid.New(), ItemType: common.SRSItemKanji, ItemID: kanjiOne, Facet: common.CardFacetMeaning, State: srs.StateLearning})
			},
			wantItems: []string{"一"}, wantQuestions: 1},
		{name: "a lesson needs a track", track: "", wantErr: exceptions.ErrLearningTrackInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tc.setup != nil {
				tc.setup(repo)
			}
			svc := NewStartLessonService(repo, NewCatalog(&fakeContent{}), NewQuestionBuilder(&fixedRand{}), zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Start(context.Background(), testUserID, &StartLessonRequest{Track: tc.track})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if len(got.Items) != len(tc.wantItems) {
				t.Fatalf("items = %+v, want %v", got.Items, tc.wantItems)
			}
			for i, item := range got.Items {
				if item.Char != tc.wantItems[i] {
					t.Errorf("item %d = %s, want %s", i, item.Char, tc.wantItems[i])
				}
			}
			if len(got.Questions) != tc.wantQuestions {
				t.Fatalf("questions = %d, want %d", len(got.Questions), tc.wantQuestions)
			}
			for _, q := range got.Questions {
				want := cardID(testUserID, common.SRSItemType(q.ItemType), q.ItemID, common.CardFacet(q.Facet))
				if q.CardID != want || q.CardState != int(srs.StateNew) || len(q.Options) == 0 {
					t.Errorf("question %+v: want derived card id %s, new state and options", q, want)
				}
			}
			stored := repo.sessions[got.Session.ID]
			if stored == nil || stored.Kind != common.SessionKindLesson || *stored.Track != common.LearningTrack(tc.track) {
				t.Errorf("stored session = %+v, want a %s lesson", stored, tc.track)
			}
			if len(repo.createdCards) != 0 {
				t.Errorf("a lesson start created %d cards, want none", len(repo.createdCards))
			}
		})
	}
}

func TestStartReviewService(t *testing.T) {
	cardFor := func(item uuid.UUID, state srs.State, stability float64, overdue time.Duration) SRSCard {
		last := testNow.Add(-overdue - time.Duration(stability*float64(24*time.Hour)))
		return SRSCard{
			ID: uuid.New(), ItemType: common.SRSItemKana, ItemID: item, Facet: common.CardFacetRecognition,
			State: state, Stability: stability, Difficulty: 5, Reps: 3, Due: testNow.Add(-overdue), LastReviewAt: &last,
		}
	}
	tests := []struct {
		name  string
		setup func(r *fakeRepo)
		want  []uuid.UUID
	}{
		{
			name: "relearning first, then the lowest recall chance",
			setup: func(r *fakeRepo) {
				r.addCard(cardFor(kanaA, srs.StateReview, 10, time.Hour))
				r.addCard(cardFor(kanaI, srs.StateReview, 2, 3*24*time.Hour))
				r.addCard(cardFor(kanaO, srs.StateRelearning, 1, time.Minute))
				r.addCard(cardFor(kanaZu, srs.StateReview, 5, -time.Hour))
			},
			want: []uuid.UUID{kanaO, kanaI, kanaA},
		},
		{
			name: "capped by the session size",
			setup: func(r *fakeRepo) {
				r.settings.SessionSize = 2
				r.addCard(cardFor(kanaA, srs.StateReview, 10, time.Hour))
				r.addCard(cardFor(kanaI, srs.StateReview, 2, 3*24*time.Hour))
				r.addCard(cardFor(kanaO, srs.StateRelearning, 1, time.Minute))
			},
			want: []uuid.UUID{kanaO, kanaI},
		},
		{
			name: "capped by the reviews left today",
			setup: func(r *fakeRepo) {
				r.reviewAnswers = 199
				r.addCard(cardFor(kanaA, srs.StateReview, 10, time.Hour))
				r.addCard(cardFor(kanaI, srs.StateReview, 2, 3*24*time.Hour))
			},
			want: []uuid.UUID{kanaI},
		},
		{
			name: "nothing due gives an empty session",
			setup: func(r *fakeRepo) {
				r.addCard(cardFor(kanaA, srs.StateReview, 10, -time.Hour))
			},
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			tc.setup(repo)
			svc := NewStartReviewService(repo, NewCatalog(&fakeContent{}), NewQuestionBuilder(&fixedRand{}), zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Start(context.Background(), testUserID, &StartReviewRequest{})
			if err != nil {
				t.Fatalf("start review: %v", err)
			}
			if len(got.Questions) != len(tc.want) {
				t.Fatalf("questions = %d, want %d", len(got.Questions), len(tc.want))
			}
			for i, q := range got.Questions {
				if q.ItemID != tc.want[i] {
					t.Errorf("question %d is for %s, want %s", i, testChars[q.ItemID], testChars[tc.want[i]])
				}
			}
			if s := repo.sessions[got.Session.ID]; s == nil || s.Kind != common.SessionKindReview || s.Track != nil {
				t.Errorf("stored session = %+v, want an all-tracks review", s)
			}
		})
	}
}
