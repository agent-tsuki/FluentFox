package progress

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
)

func TestConfusionService(t *testing.T) {
	unknown := uuid.MustParse("90000000-0000-0000-0000-000000000009")
	rows := []learn.UserConfusion{
		{ItemType: common.SRSItemKana, TargetItemID: hiraA, ChosenItemID: hiraI, Times: 4},
		{ItemType: common.SRSItemKana, TargetItemID: kataA, ChosenItemID: unknown, Times: 3},
		{ItemType: common.SRSItemKanji, TargetItemID: kanji1, ChosenItemID: kanji2, Times: 2},
		{ItemType: common.SRSItemKana, TargetItemID: hiraGa, ChosenItemID: hiraA, Times: 1},
	}
	tests := []struct {
		name string
		in   ConfusionsInput
		want []string
	}{
		{"every track, pairs outside the content skipped", ConfusionsInput{Limit: 10}, []string{"あ→い×4", "一→二×2", "が→あ×1"}},
		{"limit keeps the most frequent", ConfusionsInput{Limit: 1}, []string{"あ→い×4"}},
		{"track filter", ConfusionsInput{Track: "kanji_n5", Limit: 10}, []string{"一→二×2"}},
		{"zero limit uses the default", ConfusionsInput{}, []string{"あ→い×4", "一→二×2", "が→あ×1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.confusions = rows
			svc := NewConfusionService(repo, &fakePools{pools: testPools()}, zap.NewNop())
			got, err := svc.Get(context.Background(), testUserID, &tt.in)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			pairs := []string{}
			for _, p := range got.Pairs {
				pairs = append(pairs, fmt.Sprintf("%s→%s×%d", p.Target.Char, p.Chosen.Char, p.Times))
			}
			if !slices.Equal(pairs, tt.want) {
				t.Errorf("pairs = %q, want %q", pairs, tt.want)
			}
		})
	}
}
