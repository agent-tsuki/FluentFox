package progress

import (
	"testing"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

func TestCardLevel(t *testing.T) {
	tests := []struct {
		name      string
		state     srs.State
		stability float64
		want      common.MasteryLevel
	}{
		{"unscheduled card is new", srs.StateNew, 30, common.MasteryNew},
		{"learning step is learning whatever the stability", srs.StateLearning, 30, common.MasteryLearning},
		{"relearning step is learning", srs.StateRelearning, 10, common.MasteryLearning},
		{"review under one day is learning", srs.StateReview, 0.5, common.MasteryLearning},
		{"one day is familiar", srs.StateReview, 1, common.MasteryFamiliar},
		{"just under a week is familiar", srs.StateReview, 6.9, common.MasteryFamiliar},
		{"one week is strong", srs.StateReview, 7, common.MasteryStrong},
		{"just under three weeks is strong", srs.StateReview, 20.9, common.MasteryStrong},
		{"three weeks is mastered", srs.StateReview, 21, common.MasteryMastered},
		{"long stability is mastered", srs.StateReview, 200, common.MasteryMastered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := learn.SRSCard{State: tt.state, Stability: tt.stability}
			if got := cardLevel(&c); got != tt.want {
				t.Errorf("cardLevel = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestItemLevel(t *testing.T) {
	card := func(facet common.CardFacet, state srs.State, stability float64) *learn.SRSCard {
		return &learn.SRSCard{Facet: facet, State: state, Stability: stability}
	}
	tests := []struct {
		name     string
		itemType common.SRSItemType
		cards    []*learn.SRSCard
		want     common.MasteryLevel
	}{
		{"item without cards is new", common.SRSItemKana, nil, common.MasteryNew},
		{"kana takes its one card", common.SRSItemKana,
			[]*learn.SRSCard{card(common.CardFacetRecognition, srs.StateReview, 25)}, common.MasteryMastered},
		{"kanji takes the lower of its cards", common.SRSItemKanji, []*learn.SRSCard{
			card(common.CardFacetMeaning, srs.StateReview, 30),
			card(common.CardFacetReading, srs.StateReview, 3),
		}, common.MasteryFamiliar},
		{"kanji is mastered only when both cards are", common.SRSItemKanji, []*learn.SRSCard{
			card(common.CardFacetMeaning, srs.StateReview, 30),
			card(common.CardFacetReading, srs.StateReview, 22),
		}, common.MasteryMastered},
		{"kanji with one card counts the missing one as new", common.SRSItemKanji, []*learn.SRSCard{
			card(common.CardFacetMeaning, srs.StateReview, 30),
		}, common.MasteryNew},
		{"kanji with an unscheduled card is new", common.SRSItemKanji, []*learn.SRSCard{
			card(common.CardFacetMeaning, srs.StateReview, 30),
			card(common.CardFacetReading, srs.StateNew, 0),
		}, common.MasteryNew},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := itemLevel(tt.itemType, tt.cards); got != tt.want {
				t.Errorf("itemLevel = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestLevelCountsAndTrackProgress(t *testing.T) {
	repo := newFakeRepo()
	repo.card(common.SRSItemKana, hiraA, common.CardFacetRecognition, srs.StateReview, 30)
	repo.card(common.SRSItemKana, hiraI, common.CardFacetRecognition, srs.StateLearning, 0.1)
	pools := testPools()
	x := newProgressIndex(pools, repo.cards)

	got := x.levelCounts(pools[0])
	want := ItemsPerLevel{New: 1, Learning: 1, Mastered: 1}
	if got != want {
		t.Errorf("levelCounts = %+v, want %+v", got, want)
	}

	tests := []struct {
		name      string
		keep      func(learn.Unit) bool
		reached   func(common.MasteryLevel) bool
		wantN     int
		wantTotal int
	}{
		{"basic hiragana learned", isBasicKana, isLearned, 2, 2},
		{"hiragana mastered", anyUnit, isMastered, 1, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, total := x.trackProgress(common.LearningTrackHiragana, tt.keep, tt.reached)
			if n != tt.wantN || total != tt.wantTotal {
				t.Errorf("trackProgress = %d/%d, want %d/%d", n, total, tt.wantN, tt.wantTotal)
			}
		})
	}
}

func TestPercentOf(t *testing.T) {
	tests := []struct {
		name        string
		part, whole int
		want        float64
	}{
		{"nothing of nothing is zero", 0, 0, 0},
		{"rounds to one decimal", 1, 3, 33.3},
		{"rounds half up", 2, 3, 66.7},
		{"all is a hundred", 79, 79, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentOf(tt.part, tt.whole); got != tt.want {
				t.Errorf("percentOf(%d, %d) = %v, want %v", tt.part, tt.whole, got, tt.want)
			}
		})
	}
}
