package content

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
)

func kanaFile() KanaFile {
	tip := "tip"
	return KanaFile{
		Script: common.KanaScriptHiragana,
		Units:  []KanaFileUnit{{Slug: "a", Group: common.KanaGroupBasic, Title: "あ row", Position: 1}},
		Kana: []KanaFileEntry{
			{Slug: "a", Character: "あ", Romaji: "a", Group: common.KanaGroupBasic, Unit: "a", Position: 1,
				LookAlikes: []KanaFileLookAlike{{Slug: "o", Note: &tip}}},
			{Slug: "o", Character: "お", Romaji: "o", AltRomaji: []string{"wo"}, Group: common.KanaGroupBasic, Unit: "a", Position: 2},
		},
	}
}

func TestKanaSyncService(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(f *KanaFile)
		wantErr string
		check   func(t *testing.T, w *fakeWriter)
	}{
		{
			name: "upserts units, kana and look-alikes with the stored ids",
			check: func(t *testing.T, w *fakeWriter) {
				if len(w.kanaUnits) != 1 || len(w.kana) != 2 || len(w.kanaLookAlikes) != 1 {
					t.Fatalf("units %d kana %d look-alikes %d", len(w.kanaUnits), len(w.kana), len(w.kanaLookAlikes))
				}
				if w.kana[0].UnitID != w.ids["kana_unit:a"] || w.kana[0].AltRomaji == nil {
					t.Errorf("kana a = %+v", w.kana[0])
				}
				la := w.kanaLookAlikes[0]
				if la.KanaID != w.ids["kana:a"] || la.OtherKanaID != w.ids["kana:o"] || *la.Note != "tip" {
					t.Errorf("look-alike = %+v", la)
				}
			},
		},
		{
			name:    "unknown script is rejected before any write",
			mutate:  func(f *KanaFile) { f.Script = "romaji" },
			wantErr: "unknown script",
			check: func(t *testing.T, w *fakeWriter) {
				if len(w.kanaUnits) != 0 {
					t.Errorf("wrote %d units", len(w.kanaUnits))
				}
			},
		},
		{
			name:    "kana in an unknown unit fails the file",
			mutate:  func(f *KanaFile) { f.Kana[1].Unit = "ka" },
			wantErr: `unknown unit "ka"`,
		},
		{
			name:    "look-alike to an unknown kana fails the file",
			mutate:  func(f *KanaFile) { f.Kana[0].LookAlikes[0].Slug = "zz" },
			wantErr: `unknown look-alike "zz"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := kanaFile()
			if tc.mutate != nil {
				tc.mutate(&file)
			}
			w := newFakeWriter()

			err := NewKanaSyncService(w, zap.NewNop()).Sync(context.Background(), &file)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, w)
			}
		})
	}
}

func kanjiFile() KanjiFile {
	note := "two bars"
	return KanjiFile{
		Level:  common.JLPTLevelN5,
		Groups: []KanjiFileGroup{{Slug: "nature", Position: 1}},
		Units:  []KanjiFileUnit{{Slug: "u01", Group: "nature", Position: 1}},
		Parts:  []KanjiFilePart{{Char: "日", Name: "sun"}},
		Entries: []KanjiFileEntry{
			{
				Char: "日", Unit: "u01", Position: 1, Meanings: []string{"day"}, Strokes: 4,
				KeyReading: KanjiFileKeyReading{Type: common.ReadingTypeOn, Kana: "ニチ"},
				Kun:        []KanjiFileKun{{Kana: "ひ"}, {Kana: "か"}},
				Vocab:      []KanjiFileVocab{{Word: "日本", Reading: "にほん", Meaning: "Japan", IsKey: true}},
				Parts:      []string{"日"},
				LookAlikes: []KanjiFileLookAlike{{Char: "目", Note: &note}, {Char: "白"}},
			},
			{Char: "白", Unit: "u01", Position: 2, Meanings: []string{"white"}, Strokes: 5,
				KeyReading: KanjiFileKeyReading{Type: common.ReadingTypeKun, Kana: "しろ"}},
		},
	}
}

func TestKanjiSyncService(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(f *KanjiFile)
		wantErr string
		check   func(t *testing.T, w *fakeWriter)
	}{
		{
			name: "children get positions in file order and arrays are never nil",
			check: func(t *testing.T, w *fakeWriter) {
				if len(w.kun) != 2 || w.kun[0].Position != 1 || w.kun[1].Position != 2 {
					t.Errorf("kun = %+v", w.kun)
				}
				if len(w.vocab) != 1 || !w.vocab[0].IsKey || len(w.partLinks) != 1 {
					t.Errorf("vocab %+v links %+v", w.vocab, w.partLinks)
				}
				if w.kanji[0].Synonyms == nil || w.kanji[0].Onyomi == nil || w.kanji[0].UnitID != w.ids["kanji_unit:u01"] {
					t.Errorf("kanji = %+v", w.kanji[0])
				}
			},
		},
		{
			name: "a missing mnemonic is stored as a draft without text",
			check: func(t *testing.T, w *fakeWriter) {
				if w.kanji[1].Mnemonic != nil || !w.kanji[1].MnemonicDraft {
					t.Errorf("kanji 白 = %+v", w.kanji[1])
				}
			},
		},
		{
			name: "look-alikes to kanji not in the database are skipped, not fatal",
			check: func(t *testing.T, w *fakeWriter) {
				if len(w.kanjiLookAlikes) != 1 || w.kanjiLookAlikes[0].OtherKanjiID != w.ids["kanji:白"] {
					t.Errorf("look-alikes = %+v", w.kanjiLookAlikes)
				}
			},
		},
		{
			name:    "unit in an unknown group fails the file",
			mutate:  func(f *KanjiFile) { f.Units[0].Group = "numbers" },
			wantErr: `unknown group "numbers"`,
		},
		{
			name:    "entry with an unknown part fails the file",
			mutate:  func(f *KanjiFile) { f.Entries[0].Parts = []string{"口"} },
			wantErr: `unknown part "口"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := kanjiFile()
			if tc.mutate != nil {
				tc.mutate(&file)
			}
			w := newFakeWriter()

			err := NewKanjiSyncService(w, zap.NewNop()).Sync(context.Background(), &file)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.check(t, w)
		})
	}
}
