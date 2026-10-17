package learn

import (
	"testing"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

func TestCheckAnswer(t *testing.T) {
	hiragana := loadTestPool(common.LearningTrackHiragana)
	kanji := loadTestPool(common.LearningTrackKanjiN5)
	tests := []struct {
		name       string
		pool       *Pool
		item       uuid.UUID
		qt         common.QuestionType
		answer     AnswerRequestItem
		wantOK     bool
		wantTypo   int
		wantChosen uuid.UUID
	}{
		{"typed romaji matches the main romaji", hiragana, kanaJi, common.QuestionKanaTyped,
			AnswerRequestItem{AnswerText: ptr(" JI ")}, true, 0, uuid.Nil},
		{"typed romaji accepts an alternative spelling", hiragana, kanaJi, common.QuestionKanaTyped,
			AnswerRequestItem{AnswerText: ptr("zi")}, true, 0, uuid.Nil},
		{"empty typed answer is wrong", hiragana, kanaJi, common.QuestionKanaTyped,
			AnswerRequestItem{AnswerText: ptr("  ")}, false, 0, uuid.Nil},
		{"picked romaji of the right kana is correct", hiragana, kanaA, common.QuestionKanaMCQRomaji,
			AnswerRequestItem{ChosenOption: ptr("a")}, true, 0, uuid.Nil},
		{"picked romaji of another kana names that kana", hiragana, kanaA, common.QuestionKanaMCQRomaji,
			AnswerRequestItem{ChosenOption: ptr("o")}, false, 0, kanaO},
		{"picked kana character is compared exactly", hiragana, kanaA, common.QuestionKanaMCQKana,
			AnswerRequestItem{ChosenOption: ptr("お")}, false, 0, kanaO},
		{"a wrong typed answer records no confused item", hiragana, kanaA, common.QuestionKanaTyped,
			AnswerRequestItem{AnswerText: ptr("o")}, false, 0, uuid.Nil},
		{"meaning choice accepts a second meaning", kanji, kanjiDay, common.QuestionKanjiMeaningMCQ,
			AnswerRequestItem{ChosenOption: ptr("sun")}, true, 0, uuid.Nil},
		{"typed meaning ignores case, articles and brackets", kanji, kanjiDay, common.QuestionKanjiMeaningTyped,
			AnswerRequestItem{AnswerText: ptr("The Daytime (noun)")}, true, 0, uuid.Nil},
		{"typed meaning accepts a small typo", kanji, kanjiMiddle, common.QuestionKanjiMeaningTyped,
			AnswerRequestItem{AnswerText: ptr("insde")}, true, 1, uuid.Nil},
		{"typed meaning rejects another kanji's meaning", kanji, kanjiOne, common.QuestionKanjiMeaningTyped,
			AnswerRequestItem{AnswerText: ptr("two")}, false, 0, uuid.Nil},
		{"typed reading in hiragana matches a katakana on'yomi", kanji, kanjiOne, common.QuestionKanjiReadingTyped,
			AnswerRequestItem{AnswerText: ptr("いち")}, true, 0, uuid.Nil},
		{"typed reading accepts a kun reading with okurigana", kanji, kanjiOne, common.QuestionKanjiReadingTyped,
			AnswerRequestItem{AnswerText: ptr("ひと.つ")}, true, 0, uuid.Nil},
		{"picked reading of another kanji names that kanji", kanji, kanjiOne, common.QuestionKanjiReadingMCQ,
			AnswerRequestItem{ChosenOption: ptr("ニ")}, false, 0, kanjiTwo},
		{"listening question compares the kanji", kanji, kanjiDay, common.QuestionKanjiListen,
			AnswerRequestItem{ChosenOption: ptr("日")}, true, 0, uuid.Nil},
		{"tracing is correct unless the answer was revealed", hiragana, kanaA, common.QuestionKanaTrace,
			AnswerRequestItem{AnswerRevealed: true}, false, 0, uuid.Nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item, ok := tc.pool.Item(tc.item)
			if !ok {
				t.Fatalf("item %s not in pool", tc.item)
			}
			got := checkAnswer(tc.qt, item, tc.pool, &tc.answer)
			if got.Correct != tc.wantOK || got.TypoDistance != tc.wantTypo {
				t.Errorf("check = %+v, want correct %v typo %d", got, tc.wantOK, tc.wantTypo)
			}
			gotChosen := uuid.Nil
			if got.Chosen != nil {
				gotChosen = got.Chosen.ID
			}
			if gotChosen != tc.wantChosen {
				t.Errorf("chosen = %s, want %s", gotChosen, tc.wantChosen)
			}
		})
	}
}

func TestNormalizeReading(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"katakana becomes hiragana", "チュウ", "ちゅう"},
		{"separators are dropped", " ひと.つ ", "ひとつ"},
		{"hiragana is kept", "にち", "にち"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeReading(tc.in); got != tc.want {
				t.Errorf("normalizeReading(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
