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

func TestSummaryServiceGet(t *testing.T) {
	tests := []struct {
		name     string
		track    string
		setup    func(r *fakeRepo)
		want     LearnSummary
		wantUnit string
		wantErr  error
	}{
		{
			name:  "a new user has everything ahead of them",
			track: "",
			want:  LearnSummary{NewLeftToday: 7 + 5, Goal: LearnGoal{GoalAnswers: 20}},
			// 6 hiragana + 1 katakana fit in the kana limit of 15; 5 kanji in 10.
			wantUnit: "a",
		},
		{
			name:  "due counts split into now and later today",
			track: "hiragana",
			setup: func(r *fakeRepo) {
				r.addCard(SRSCard{ID: uuid.New(), ItemType: common.SRSItemKana, ItemID: kanaA, Facet: common.CardFacetRecognition,
					State: srs.StateReview, Due: testNow.Add(-time.Hour)})
				r.addCard(SRSCard{ID: uuid.New(), ItemType: common.SRSItemKana, ItemID: kanaI, Facet: common.CardFacetRecognition,
					State: srs.StateReview, Due: testNow.Add(5 * time.Hour)})
				r.addCard(SRSCard{ID: uuid.New(), ItemType: common.SRSItemKana, ItemID: kanaO, Facet: common.CardFacetRecognition,
					State: srs.StateReview, Due: testNow.Add(48 * time.Hour)})
				r.learned = 3
			},
			want:     LearnSummary{DueNow: 1, DueToday: 2, NewLeftToday: 3, Learned: 3, Goal: LearnGoal{GoalAnswers: 20}},
			wantUnit: "za",
		},
		{
			name:  "new items left are capped by what was introduced today",
			track: "hiragana",
			setup: func(r *fakeRepo) { r.introducedToday[common.SRSItemKana] = 13 },
			want:  LearnSummary{NewLeftToday: 2, Goal: LearnGoal{GoalAnswers: 20}}, wantUnit: "a",
		},
		{
			name:  "hiragana and katakana share the kana limit",
			track: "",
			setup: func(r *fakeRepo) {
				r.settings.NewKanaPerDay = 6
				r.settings.NewKanjiPerDay = 0
			},
			want: LearnSummary{NewLeftToday: 6, Goal: LearnGoal{GoalAnswers: 20}}, wantUnit: "a",
		},
		{
			name:  "goal progress comes from today's activity",
			track: "kanji_n5",
			setup: func(r *fakeRepo) {
				r.activity["2026-03-10"] = &DailyActivity{ReviewsDone: 21, GoalMet: true}
			},
			want:     LearnSummary{NewLeftToday: 5, Goal: LearnGoal{AnswersToday: 21, GoalAnswers: 20, Met: true}},
			wantUnit: "u01",
		},
		{name: "an unknown track is rejected", track: "romaji", wantErr: exceptions.ErrLearningTrackInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tc.setup != nil {
				tc.setup(repo)
			}
			svc := NewSummaryService(repo, NewCatalog(&fakeContent{}), zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Get(context.Background(), testUserID, tc.track)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			unit := ""
			if got.NextUnit != nil {
				unit = got.NextUnit.Slug
			}
			got.NextUnit = nil
			if got != tc.want || unit != tc.wantUnit {
				t.Errorf("summary = %+v next %q, want %+v next %q", got, unit, tc.want, tc.wantUnit)
			}
		})
	}
}
