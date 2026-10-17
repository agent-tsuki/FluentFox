package progress

import (
	"context"
	"math"
	"slices"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

func TestWeaknessScore(t *testing.T) {
	tests := []struct {
		name    string
		signals weakSignals
		want    float64
	}{
		{"strong card scores zero", weakSignals{Recall: 1, TimeRatio: 1}, 0},
		{"recall chance weighs 0.4", weakSignals{Recall: 0.5, TimeRatio: 1}, 0.2},
		{"lapse rate weighs 0.3", weakSignals{Recall: 1, LapseRate: 1.0 / 3, TimeRatio: 1}, 0.1},
		{"half again the usual time adds half the time weight", weakSignals{Recall: 1, TimeRatio: 1.5}, 0.1},
		{"time counts in full from twice the usual", weakSignals{Recall: 1, TimeRatio: 3}, 0.2},
		{"faster than usual adds nothing", weakSignals{Recall: 1, TimeRatio: 0.5}, 0},
		{"no timed answers adds nothing", weakSignals{Recall: 1, TimeRatio: 0}, 0},
		{"confusion share weighs 0.1", weakSignals{Recall: 1, TimeRatio: 1, ConfusionShare: 0.5}, 0.05},
		{"everything at its worst scores one", weakSignals{Recall: 0, LapseRate: 1, TimeRatio: 2, ConfusionShare: 1}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weaknessScore(tt.signals); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("weaknessScore = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWeakReasons(t *testing.T) {
	avg := 3200
	tests := []struct {
		name  string
		facts weakFacts
		want  []string
	}{
		{"forgotten several times", weakFacts{Lapses: 3, Recall: 0.9}, []string{"Forgotten 3 times"}},
		{"forgotten once", weakFacts{Lapses: 1, Recall: 0.9}, []string{"Forgotten once"}},
		{"slow against the usual time", weakFacts{Recall: 0.9, TimeRatio: 2, AvgMs: &avg, UsualMs: 1600},
			[]string{"Slow: 3.2s (your usual 1.6s)"}},
		{"a little slower is not slow", weakFacts{Recall: 0.9, TimeRatio: 1.2, AvgMs: &avg, UsualMs: 2700},
			[]string{"Recall chance 90%"}},
		{"mixed up", weakFacts{Recall: 0.9, Confusion: &topConfusion{Char: "め", Times: 4}}, []string{"Mixed up with め 4×"}},
		{"low recall is a reason of its own", weakFacts{Lapses: 2, Recall: 0.45}, []string{"Forgotten 2 times", "Recall chance 45%"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weakReasons(&tt.facts); !slices.Equal(got, tt.want) {
				t.Errorf("weakReasons = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWeakSuggestion(t *testing.T) {
	pools := testPools()
	pool, item, _ := pools.Find(common.SRSItemKana, hiraA)
	ref := cardRef{Pool: pool, Item: item}
	tests := []struct {
		name  string
		facts weakFacts
		want  PracticeSuggestion
	}{
		{"mix-up drills the pair", weakFacts{TimeRatio: 3, Confusion: &topConfusion{Char: "お", Times: 2}},
			PracticeSuggestion{Track: "hiragana", Mode: "lookalike", Pair: "あ:お"}},
		{"slow card gets a speed round", weakFacts{TimeRatio: 2}, PracticeSuggestion{Track: "hiragana", Mode: "speed"}},
		{"otherwise its unit", weakFacts{TimeRatio: 1}, PracticeSuggestion{Track: "hiragana", Mode: "unit", Unit: "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weakSuggestion(&ref, &tt.facts); got != tt.want {
				t.Errorf("weakSuggestion = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// weakFixture: あ is forgotten often, slow and mixed up with い; い is
// strong; が has too few reps; 一 was never scheduled.
func weakFixture() *fakeRepo {
	repo := newFakeRepo()
	recent := testNow.Add(-time.Hour)
	longAgo := testNow.Add(-10 * 24 * time.Hour)
	a := repo.card(common.SRSItemKana, hiraA, common.CardFacetRecognition, srs.StateReview, 2, func(c *learn.SRSCard) {
		c.Reps, c.Lapses, c.LastReviewAt = 6, 3, &longAgo
	})
	i := repo.card(common.SRSItemKana, hiraI, common.CardFacetRecognition, srs.StateReview, 30, func(c *learn.SRSCard) {
		c.Reps, c.LastReviewAt = 5, &recent
	})
	repo.card(common.SRSItemKana, hiraGa, common.CardFacetRecognition, srs.StateReview, 1, func(c *learn.SRSCard) { c.Reps = 1 })
	repo.card(common.SRSItemKanji, kanji1, common.CardFacetMeaning, srs.StateNew, 0, func(c *learn.SRSCard) { c.Reps = 4 })
	// Usual kana MCQ time: (16000 + 16000) ms / 20 answers = 1.6 s; あ takes 3.2 s.
	repo.stats = []learn.CardAnswerStats{
		stat(a, common.QuestionKanaMCQRomaji, 8, 5, 6, 3, 5, 16000),
		stat(i, common.QuestionKanaMCQRomaji, 15, 15, 5, 5, 15, 16000),
	}
	repo.confusions = []learn.UserConfusion{
		{ItemType: common.SRSItemKana, TargetItemID: hiraA, ChosenItemID: hiraI, Times: 4},
		{ItemType: common.SRSItemKana, TargetItemID: hiraA, ChosenItemID: hiraGa, Times: 1},
	}
	return repo
}

func TestWeakItemsService(t *testing.T) {
	tests := []struct {
		name      string
		in        WeakItemsInput
		wantChars []string
	}{
		{"weakest first, skipping new and barely reviewed cards", WeakItemsInput{Limit: 20}, []string{"あ", "い"}},
		{"limit keeps the weakest", WeakItemsInput{Limit: 1}, []string{"あ"}},
		{"track filter", WeakItemsInput{Track: "kanji_n5", Limit: 20}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewWeakItemsService(weakFixture(), &fakePools{pools: testPools()}, zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Get(context.Background(), testUserID, &tt.in)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			chars := []string{}
			for _, it := range got.Items {
				chars = append(chars, it.Char)
			}
			if !slices.Equal(chars, tt.wantChars) {
				t.Errorf("chars = %q, want %q", chars, tt.wantChars)
			}
		})
	}
}

func TestWeakItemsServiceWeakestItemDetails(t *testing.T) {
	svc := NewWeakItemsService(weakFixture(), &fakePools{pools: testPools()}, zap.NewNop())
	svc.now = func() time.Time { return testNow }
	got, err := svc.Get(context.Background(), testUserID, &WeakItemsInput{Limit: 1})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	a := got.Items[0]
	// Ten days after the last review at 2 days' stability, recall is low too.
	wantReasons := []string{"Forgotten 3 times", "Slow: 3.2s (your usual 1.6s)", "Mixed up with い 4×", "Recall chance 76%"}
	if !slices.Equal(a.Reasons, wantReasons) {
		t.Errorf("reasons = %q, want %q", a.Reasons, wantReasons)
	}
	if a.Suggestion != (PracticeSuggestion{Track: "hiragana", Mode: "lookalike", Pair: "あ:い"}) {
		t.Errorf("suggestion = %+v", a.Suggestion)
	}
	if a.LapseRate != 0.5 || a.TimeRatio != 2 || a.ConfusionShare != 0.5 {
		t.Errorf("signals = lapse %v, time %v, confusion %v; want 0.5, 2, 0.5", a.LapseRate, a.TimeRatio, a.ConfusionShare)
	}
	if a.Recall >= common.WEAK_LOW_RECALL {
		t.Errorf("recall = %v, want below %v", a.Recall, common.WEAK_LOW_RECALL)
	}
}
