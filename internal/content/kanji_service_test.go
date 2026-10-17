package content

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/exceptions"
)

// kanjiFixture is a two-unit N5 level: 一 二 in u01, 日 in u02, plus 目 at N4.
func kanjiFixture() *fakeKanjiReader {
	groupID, u1, u2, u3 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ichi, ni, hi, me := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mnemonic := "one line"
	freq := 2
	okuri := "つ"
	prefix := common.KunAffixPrefix
	note := "目 has two inner bars"
	return &fakeKanjiReader{
		groups: []KanjiGroup{{ID: groupID, Level: common.JLPTLevelN5, Slug: "numbers", Title: "Numbers", Position: 1}},
		units: []KanjiUnit{
			{ID: u1, GroupID: groupID, Level: common.JLPTLevelN5, Slug: "u01", Title: "One, two", Position: 1},
			{ID: u2, GroupID: groupID, Level: common.JLPTLevelN5, Slug: "u02", Title: "Sun", Position: 2},
			{ID: u3, Level: common.JLPTLevelN4, Slug: "u01", Position: 1},
		},
		kanji: []Kanji{
			{
				ID: ichi, Char: "一", Level: common.JLPTLevelN5, UnitID: u1, Position: 1,
				Meanings: database.TextArray{"one"}, Synonyms: database.TextArray{"single"}, Onyomi: database.TextArray{"イチ", "イツ"},
				KeyReadingType: common.ReadingTypeOn, KeyReading: "イチ", Mnemonic: &mnemonic, MnemonicDraft: true, Strokes: 1, Freq: &freq,
			},
			{ID: ni, Char: "二", Level: common.JLPTLevelN5, UnitID: u1, Position: 2, Meanings: database.TextArray{"two"}, Strokes: 2},
			{ID: hi, Char: "日", Level: common.JLPTLevelN5, UnitID: u2, Position: 3, Meanings: database.TextArray{"day"}, Strokes: 4},
			{ID: me, Char: "目", Level: common.JLPTLevelN4, UnitID: u3, Position: 1, Meanings: database.TextArray{"eye"}, Strokes: 5},
		},
		kun: []KanjiKunReading{
			{KanjiID: ichi, Kana: "ひと", Affix: &prefix, Position: 1},
			{KanjiID: ichi, Kana: "ひと", Okurigana: &okuri, Position: 2},
		},
		parts: []KanjiPartRow{
			{KanjiID: ichi, Char: "一", Name: "one"},
			{KanjiID: ni, Char: "二", Name: "two"},
			{KanjiID: hi, Char: "日", Name: "sun"},
			{KanjiID: hi, Char: "一", Name: "one"},
		},
		vocab:      []KanjiVocab{{KanjiID: ichi, Word: "一つ", Reading: "ひとつ", Meaning: "one", IsKey: true, Position: 1}},
		lookAlikes: []KanjiLookAlikeRow{{KanjiID: hi, OtherChar: "目", Note: &note}},
	}
}

func TestKanjiContentServiceLevelNotServed(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{name: "upper-case level id is not found", level: "N5"},
		{name: "level without content in v1 is not found", level: "n4"},
		{name: "unknown level is not found", level: "n9"},
		{name: "empty level is not found", level: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewKanjiContentService(kanjiFixture(), "", zap.NewNop())

			_, indexErr := svc.GetLevelIndex(context.Background(), tc.level)
			_, detailErr := svc.GetLevelDetail(context.Background(), tc.level)

			if !errors.Is(indexErr, exceptions.ErrKanjiLevelNotFound) || !errors.Is(detailErr, exceptions.ErrKanjiLevelNotFound) {
				t.Fatalf("errors = %v, %v; want ErrKanjiLevelNotFound", indexErr, detailErr)
			}
		})
	}
}

func TestKanjiContentServiceGetLevelIndex(t *testing.T) {
	tests := []struct {
		name  string
		check func(t *testing.T, got KanjiLevelIndex, repo *fakeKanjiReader)
	}{
		{
			name: "units list their kanji and only the parts no earlier unit used",
			check: func(t *testing.T, got KanjiLevelIndex, _ *fakeKanjiReader) {
				want := []KanjiUnitDTO{
					{ID: "u01", Title: "One, two", Kanji: []string{"一", "二"}, NewParts: []string{"一", "二"}},
					{ID: "u02", Title: "Sun", Kanji: []string{"日"}, NewParts: []string{"日"}},
				}
				if !reflect.DeepEqual(got.Units[:2], want) {
					t.Errorf("units = %+v, want %+v", got.Units, want)
				}
			},
		},
		{
			name: "groups list their unit ids in order",
			check: func(t *testing.T, got KanjiLevelIndex, _ *fakeKanjiReader) {
				if !reflect.DeepEqual(got.Groups[0].UnitIDs[:2], []string{"u01", "u02"}) {
					t.Errorf("group unitIds = %v", got.Groups[0].UnitIDs)
				}
			},
		},
		{
			name: "rows add synonyms to meanings and list each kun stem once",
			check: func(t *testing.T, got KanjiLevelIndex, _ *fakeKanjiReader) {
				row := got.Rows[0]
				if row.Level != "n5" || row.UnitID != "u01" || row.Meaning != "one" || row.KeyReading != "イチ" {
					t.Errorf("row = %+v", row)
				}
				if !reflect.DeepEqual(row.Meanings, []string{"one", "single"}) {
					t.Errorf("meanings = %v", row.Meanings)
				}
				if !reflect.DeepEqual(row.Readings, []string{"イチ", "イツ", "ひと"}) {
					t.Errorf("readings = %v", row.Readings)
				}
			},
		},
		{
			name: "the index does not load vocab or look-alikes",
			check: func(t *testing.T, _ KanjiLevelIndex, repo *fakeKanjiReader) {
				if slices.Contains(repo.calls, "vocab") || slices.Contains(repo.calls, "lookAlikes") {
					t.Errorf("calls = %v", repo.calls)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := kanjiFixture()
			repo.kanji = repo.kanji[:3]
			svc := NewKanjiContentService(repo, "https://cdn", zap.NewNop())

			got, err := svc.GetLevelIndex(context.Background(), "n5")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Level != "n5" || got.StrokesBaseURL != "https://cdn" {
				t.Errorf("level/url = %q %q", got.Level, got.StrokesBaseURL)
			}
			tc.check(t, got, repo)
		})
	}
}

func TestKanjiContentServiceGetEntry(t *testing.T) {
	tests := []struct {
		name    string
		char    string
		wantErr error
		check   func(t *testing.T, got KanjiEntryResponse)
	}{
		{name: "unknown kanji is not found", char: "龍", wantErr: exceptions.ErrKanjiNotFound},
		{name: "kanji of a level not served is not found", char: "目", wantErr: exceptions.ErrKanjiNotFound},
		{
			name: "entry carries kun readings, vocab, mnemonic and the stroke url",
			char: "一",
			check: func(t *testing.T, got KanjiEntryResponse) {
				if got.JLPT != "n5" || got.UnitID != "u01" || got.StrokesBaseURL != "https://cdn" {
					t.Errorf("entry = %+v", got)
				}
				if len(got.Kun) != 2 || *got.Kun[0].Affix != "prefix" || *got.Kun[1].Okurigana != "つ" {
					t.Errorf("kun = %+v", got.Kun)
				}
				if got.Mnemonic == nil || got.Mnemonic.Text != "one line" || !got.Mnemonic.Draft {
					t.Errorf("mnemonic = %+v", got.Mnemonic)
				}
				if len(got.Vocab) != 1 || !got.Vocab[0].IsKey || got.KeyReading.Type != "on" {
					t.Errorf("vocab/key = %+v %+v", got.Vocab, got.KeyReading)
				}
			},
		},
		{
			name: "entry without a mnemonic omits it and keeps look-alikes",
			char: "日",
			check: func(t *testing.T, got KanjiEntryResponse) {
				if got.Mnemonic != nil {
					t.Errorf("mnemonic = %+v, want nil", got.Mnemonic)
				}
				if len(got.LookAlikes) != 1 || got.LookAlikes[0].Char != "目" {
					t.Errorf("lookAlikes = %+v", got.LookAlikes)
				}
				if !reflect.DeepEqual(got.Parts, []KanjiPartDTO{{Char: "日", Name: "sun"}, {Char: "一", Name: "one"}}) {
					t.Errorf("parts = %+v", got.Parts)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewKanjiContentService(kanjiFixture(), "https://cdn", zap.NewNop())

			got, err := svc.GetEntry(context.Background(), tc.char)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.check(t, got)
		})
	}
}

func TestKanjiContentServiceGetLevelDetail(t *testing.T) {
	repo := kanjiFixture()
	repo.kanji = repo.kanji[:3]
	svc := NewKanjiContentService(repo, "", zap.NewNop())

	got, err := svc.GetLevelDetail(context.Background(), "n5")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	chars := []string{}
	for _, e := range got.Entries {
		chars = append(chars, e.Char)
	}
	if !reflect.DeepEqual(chars, []string{"一", "二", "日"}) {
		t.Errorf("entries = %v, want course order", chars)
	}
	if got.Entries[1].Kun == nil || got.Entries[1].Vocab == nil || got.Entries[1].Parts == nil {
		t.Error("empty lists must be [] not null")
	}
}
