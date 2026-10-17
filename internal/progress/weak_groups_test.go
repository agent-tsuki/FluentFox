package progress

import (
	"context"
	"slices"
	"testing"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

// weakGroupsFixture: the user misses 12 of 45 reviews (27%). Katakana
// ア misses 6 of 10; が is answered in 4 s against a usual 1.75 s; 一 has
// too few reviews to judge on its own but counts in the average.
func weakGroupsFixture() *fakeRepo {
	repo := newFakeRepo()
	qt := common.QuestionKanaMCQRomaji
	a := repo.card(common.SRSItemKana, hiraA, common.CardFacetRecognition, srs.StateReview, 10)
	i := repo.card(common.SRSItemKana, hiraI, common.CardFacetRecognition, srs.StateReview, 10)
	ga := repo.card(common.SRSItemKana, hiraGa, common.CardFacetRecognition, srs.StateReview, 10)
	ka := repo.card(common.SRSItemKana, kataA, common.CardFacetRecognition, srs.StateReview, 10)
	k1 := repo.card(common.SRSItemKanji, kanji1, common.CardFacetMeaning, srs.StateReview, 10)
	repo.stats = []learn.CardAnswerStats{
		stat(a, qt, 10, 9, 10, 9, 10, 10000),
		stat(i, qt, 10, 10, 10, 10, 10, 10000),
		stat(ga, qt, 10, 10, 10, 10, 10, 40000),
		stat(ka, qt, 10, 4, 10, 4, 10, 10000),
		stat(k1, common.QuestionKanjiMeaningMCQ, 5, 0, 5, 0, 0, 0),
	}
	return repo
}

func TestWeakGroupsService(t *testing.T) {
	tests := []struct {
		name  string
		track string
		repo  func() *fakeRepo
		want  []string
	}{
		{"misses and slowness, most missed first", "", weakGroupsFixture,
			[]string{"katakana/a: Missed 60% of reviews (your average 27%)",
				"hiragana/ga: Slow: 4.0s per answer (your usual 1.8s)",
				"hiragana/dakuten: Slow: 4.0s per answer (your usual 1.8s)"}},
		{"track filter still compares with the overall rate", "katakana", weakGroupsFixture,
			[]string{"katakana/a: Missed 60% of reviews (your average 27%)"}},
		{"too few reviews lists nothing", "", func() *fakeRepo {
			r := weakGroupsFixture()
			for j := range r.stats {
				r.stats[j].Reviews = min(r.stats[j].Reviews, common.WEAK_GROUP_MIN_REVIEWS-1)
			}
			return r
		}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewWeakGroupsService(tt.repo(), &fakePools{pools: testPools()}, zap.NewNop())
			got, err := svc.Get(context.Background(), testUserID, tt.track)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			lines := []string{}
			for _, g := range got.Groups {
				lines = append(lines, g.Track+"/"+g.Slug+": "+g.Reason)
			}
			if !slices.Equal(lines, tt.want) {
				t.Errorf("groups =\n%q\nwant\n%q", lines, tt.want)
			}
		})
	}
}

func TestWeakGroupsSuggestionAndKind(t *testing.T) {
	svc := NewWeakGroupsService(weakGroupsFixture(), &fakePools{pools: testPools()}, zap.NewNop())
	got, err := svc.Get(context.Background(), testUserID, "hiragana")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	tests := []struct {
		name       string
		group      WeakGroup
		wantKind   string
		wantUnitID bool
		want       PracticeSuggestion
	}{
		{"a unit is practised as its unit", got.Groups[0], "unit", true, PracticeSuggestion{Track: "hiragana", Mode: "unit", Unit: "ga"}},
		{"a kana group gets the weak drill", got.Groups[1], "kana_group", false, PracticeSuggestion{Track: "hiragana", Mode: "weak"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.group.Kind != tt.wantKind || (tt.group.UnitID != nil) != tt.wantUnitID || tt.group.Suggestion != tt.want {
				t.Errorf("group = %+v", tt.group)
			}
		})
	}
}
