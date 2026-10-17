package content

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestKanaContentServiceGetCourse(t *testing.T) {
	rowA, rowCombo := uuid.New(), uuid.New()
	kaID, kyaID, aID := uuid.New(), uuid.New(), uuid.New()
	tip := "a tip"
	units := []KanaUnit{
		{ID: rowA, Script: common.KanaScriptHiragana, GroupKey: common.KanaGroupBasic, Slug: "a", Title: "あ row", Summary: "vowels", Position: 1},
		{ID: rowCombo, Script: common.KanaScriptHiragana, GroupKey: common.KanaGroupCombo, Slug: "combo-1", Title: "Combinations I", Position: 2},
	}
	strokes := 3
	kana := []Kana{
		{ID: aID, Slug: "a", Character: "あ", Romaji: "a", GroupKey: common.KanaGroupBasic, UnitID: rowA, Position: 1, StrokeCount: &strokes},
		{ID: kaID, Slug: "o", Character: "お", Romaji: "o", AltRomaji: database.TextArray{"wo"}, GroupKey: common.KanaGroupBasic, UnitID: rowA, Position: 2},
		{ID: kyaID, Slug: "kya", Character: "きゃ", Romaji: "kya", GroupKey: common.KanaGroupCombo, UnitID: rowCombo, Position: 3},
	}
	lookAlikes := []KanaLookAlikeRow{{KanaID: aID, OtherSlug: "o", OtherChar: "お", Note: &tip}}
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		script  string
		repo    *fakeKanaReader
		wantErr error
		check   func(t *testing.T, got KanaCourseResponse)
	}{
		{
			name:    "unknown script is not found",
			script:  "romaji",
			repo:    &fakeKanaReader{},
			wantErr: exceptions.ErrKanaScriptNotFound,
		},
		{
			name:    "upper-case script is not found",
			script:  "Hiragana",
			repo:    &fakeKanaReader{},
			wantErr: exceptions.ErrKanaScriptNotFound,
		},
		{
			name:    "repository error is returned",
			script:  "hiragana",
			repo:    &fakeKanaReader{err: dbErr},
			wantErr: dbErr,
		},
		{
			name:   "builds groups, units and kana in path order",
			script: "hiragana",
			repo:   &fakeKanaReader{units: units, kana: kana, lookAlikes: lookAlikes},
			check: func(t *testing.T, got KanaCourseResponse) {
				wantGroups := []KanaGroupDTO{
					{ID: "basic", Title: "Basic 46", Summary: "The core hiragana, あ to ん.", UnitIDs: []string{"a"}},
					{ID: "combo", Title: "Combinations", Summary: "Small ゃゅょ combinations.", UnitIDs: []string{"combo-1"}},
				}
				if !reflect.DeepEqual(got.Groups, wantGroups) {
					t.Errorf("groups = %+v, want %+v", got.Groups, wantGroups)
				}
				if !reflect.DeepEqual(got.Units[0].KanaIDs, []string{"a", "o"}) {
					t.Errorf("unit a kanaIds = %v", got.Units[0].KanaIDs)
				}
				if got.Script != "hiragana" || got.StrokesBaseURL != "https://cdn/strokes" {
					t.Errorf("script/url = %q %q", got.Script, got.StrokesBaseURL)
				}
			},
		},
		{
			name:   "combo kana are compound in the chart and carry their look-alikes",
			script: "hiragana",
			repo:   &fakeKanaReader{units: units, kana: kana, lookAlikes: lookAlikes},
			check: func(t *testing.T, got KanaCourseResponse) {
				kya := got.Kana[2]
				if kya.Type != "compound" || !kya.IsCompound || kya.GroupID != "combo" || kya.UnitID != "combo-1" {
					t.Errorf("kya = %+v", kya)
				}
				a := got.Kana[0]
				wantLA := []KanaLookAlikeDTO{{ID: "o", Char: "お", Note: &tip}}
				if !reflect.DeepEqual(a.LookAlikes, wantLA) || a.Type != "basic" || a.IsCompound || *a.StrokeCount != 3 {
					t.Errorf("a = %+v", a)
				}
				if a.AltRomaji == nil || got.Kana[2].LookAlikes == nil {
					t.Error("empty lists must be [] not null")
				}
				if !reflect.DeepEqual(got.Kana[1].AltRomaji, []string{"wo"}) {
					t.Errorf("o altRomaji = %v", got.Kana[1].AltRomaji)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewKanaContentService(tc.repo, "https://cdn/strokes", zap.NewNop())

			got, err := svc.GetCourse(context.Background(), tc.script)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.repo.gotScript != common.KanaScriptHiragana {
				t.Errorf("repo script = %q", tc.repo.gotScript)
			}
			tc.check(t, got)
		})
	}
}
