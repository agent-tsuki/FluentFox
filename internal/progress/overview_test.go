package progress

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

func overviewFixture() *fakeRepo {
	repo := newFakeRepo()
	a := repo.card(common.SRSItemKana, hiraA, common.CardFacetRecognition, srs.StateReview, 30)
	repo.card(common.SRSItemKana, hiraI, common.CardFacetRecognition, srs.StateLearning, 0.2, dueAt(testNow.Add(-time.Hour)))
	k1 := repo.card(common.SRSItemKanji, kanji1, common.CardFacetMeaning, srs.StateReview, 30)
	repo.card(common.SRSItemKanji, kanji1, common.CardFacetReading, srs.StateReview, 10)
	repo.card(common.SRSItemKanji, kanji2, common.CardFacetMeaning, srs.StateReview, 25)
	repo.stats = []learn.CardAnswerStats{
		stat(a, common.QuestionKanaMCQRomaji, 12, 11, 10, 9, 10, 15000),
		stat(k1, common.QuestionKanjiMeaningMCQ, 10, 7, 10, 7, 7, 21000),
	}
	repo.activity = []learn.DailyActivity{
		{LocalDate: date("2026-03-10"), ReviewsDone: 12, CorrectCount: 10, NewLearned: 3, ActiveMs: 60_000},
	}
	repo.goalDates = []time.Time{date("2026-03-08"), date("2026-03-09")}
	repo.introduced[common.SRSItemKana] = 5
	repo.introduced[common.SRSItemKanji] = 12
	repo.reviewCount = 50
	return repo
}

func ptr[T any](v T) *T { return &v }

func TestOverviewServiceTracks(t *testing.T) {
	repo := overviewFixture()
	svc := NewOverviewService(repo, &fakePools{pools: testPools()}, zap.NewNop())
	svc.now = func() time.Time { return testNow }
	got, err := svc.Get(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if want := testNow.Add(-common.RETENTION_WINDOW); !repo.statsFilter.Since.Equal(want) {
		t.Errorf("stats since %v, want %v", repo.statsFilter.Since, want)
	}
	tests := []struct {
		name string
		got  TrackOverview
		want TrackOverview
	}{
		{"hiragana", got.Tracks[0], TrackOverview{
			Track: "hiragana", TotalItems: 3, ItemsPerLevel: ItemsPerLevel{New: 1, Learning: 1, Mastered: 1},
			MasteredPct: 33.3, Retention30d: ptr(0.9), AvgAnswerMs: ptr(1500),
		}},
		{"katakana without answers", got.Tracks[1], TrackOverview{
			Track: "katakana", TotalItems: 1, ItemsPerLevel: ItemsPerLevel{New: 1},
		}},
		{"kanji takes the lower card and a missing card is new", got.Tracks[2], TrackOverview{
			Track: "kanji_n5", TotalItems: 2, ItemsPerLevel: ItemsPerLevel{New: 1, Strong: 1},
			MasteredPct: 0, Retention30d: ptr(0.7), AvgAnswerMs: ptr(3000),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, w := tt.got, tt.want
			if g.Track != w.Track || g.TotalItems != w.TotalItems || g.ItemsPerLevel != w.ItemsPerLevel || g.MasteredPct != w.MasteredPct {
				t.Errorf("track = %+v, want %+v", g, w)
			}
			if !samePtr(g.Retention30d, w.Retention30d) || !samePtr(g.AvgAnswerMs, w.AvgAnswerMs) {
				t.Errorf("retention %v avg %v, want %v %v", deref(g.Retention30d), deref(g.AvgAnswerMs), deref(w.Retention30d), deref(w.AvgAnswerMs))
			}
			var fromItems ItemsPerLevel
			for _, item := range g.Items {
				fromItems.add(common.MasteryLevel(item.Level))
			}
			if len(g.Items) != w.TotalItems || fromItems != w.ItemsPerLevel {
				t.Errorf("items %+v do not match the level counts %+v", g.Items, w.ItemsPerLevel)
			}
		})
	}
}

func TestOverviewServiceTodayStreakAndRetention(t *testing.T) {
	tests := []struct {
		name          string
		showRetention bool
		wantRetention *RetentionInfo
	}{
		{"retention shown against the target", true, &RetentionInfo{Actual: ptr(0.8), Target: 0.9}},
		{"retention hidden in settings", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := overviewFixture()
			repo.settings.ShowRetention = tt.showRetention
			svc := NewOverviewService(repo, &fakePools{pools: testPools()}, zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Get(context.Background(), testUserID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			wantToday := TodayStats{
				Answers: 12, Correct: 10, ActiveMs: 60_000, NewLearned: 3, NewKanaLeft: 10, NewKanjiLeft: 0,
				ReviewsLeft: 150, GoalAnswers: 20, DueNow: 1,
			}
			if got.Today != wantToday {
				t.Errorf("today = %+v, want %+v", got.Today, wantToday)
			}
			if got.Streak != (StreakInfo{Current: 2, Longest: 2}) {
				t.Errorf("streak = %+v", got.Streak)
			}
			if (got.Retention == nil) != (tt.wantRetention == nil) {
				t.Fatalf("retention = %+v, want %+v", got.Retention, tt.wantRetention)
			}
			if got.Retention != nil && (!samePtr(got.Retention.Actual, tt.wantRetention.Actual) || got.Retention.Target != tt.wantRetention.Target) {
				t.Errorf("retention = %v/%v, want %v/%v", deref(got.Retention.Actual), got.Retention.Target, deref(tt.wantRetention.Actual), tt.wantRetention.Target)
			}
		})
	}
}

func samePtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func deref[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
