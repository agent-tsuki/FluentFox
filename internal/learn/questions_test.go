package learn

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

func TestQuestionBuilderChoosesType(t *testing.T) {
	hiragana := loadTestPool(common.LearningTrackHiragana)
	kanji := loadTestPool(common.LearningTrackKanjiN5)
	stableReview := QuestionCard{State: srs.StateReview, Stability: common.TYPED_MIN_STABILITY_DAYS}
	tests := []struct {
		name  string
		pool  *Pool
		item  uuid.UUID
		card  QuestionCard
		mode  common.AnswerMode
		rolls []int
		want  common.QuestionType
	}{
		{"a new kana card gets romaji choice", hiragana, kanaI, QuestionCard{State: srs.StateNew}, common.AnswerModeMixed,
			[]int{0}, common.QuestionKanaMCQRomaji},
		{"a new kana card may get kana choice", hiragana, kanaI, QuestionCard{State: srs.StateNew}, common.AnswerModeMixed,
			[]int{1}, common.QuestionKanaMCQKana},
		{"a new card never gets a variety question", hiragana, kanaA, QuestionCard{State: srs.StateNew}, common.AnswerModeTyped,
			[]int{0, 0}, common.QuestionKanaMCQRomaji},
		{"a stable review card is typed in mixed mode", hiragana, kanaI, stableReview, common.AnswerModeMixed,
			nil, common.QuestionKanaTyped},
		{"a young review card stays multiple choice in mixed mode", hiragana, kanaI,
			QuestionCard{State: srs.StateReview, Stability: 1}, common.AnswerModeMixed, []int{1, 0}, common.QuestionKanaMCQRomaji},
		{"a learning card is typed in typed mode", hiragana, kanaI, QuestionCard{State: srs.StateLearning}, common.AnswerModeTyped,
			nil, common.QuestionKanaTyped},
		{"choice mode never types", hiragana, kanaI, stableReview, common.AnswerModeChoice, []int{1, 0}, common.QuestionKanaMCQRomaji},
		{"one in four kana questions is a look-alike when the kana has some", hiragana, kanaA, stableReview,
			common.AnswerModeMixed, []int{0, 0}, common.QuestionKanaLookalike},
		{"a kana without look-alikes gets listening instead", hiragana, kanaI, stableReview,
			common.AnswerModeMixed, []int{0}, common.QuestionKanaListen},
		{"a kanji meaning card gets meaning choice", kanji, kanjiDay, QuestionCard{State: srs.StateLearning, Facet: common.CardFacetMeaning},
			common.AnswerModeMixed, nil, common.QuestionKanjiMeaningMCQ},
		{"a stable kanji reading card is typed", kanji, kanjiDay,
			QuestionCard{State: srs.StateReview, Stability: 10, Facet: common.CardFacetReading},
			common.AnswerModeMixed, nil, common.QuestionKanjiReadingTyped},
		{"a kanji reading variety question is listening", kanji, kanjiDay,
			QuestionCard{State: srs.StateReview, Stability: 10, Facet: common.CardFacetReading},
			common.AnswerModeMixed, []int{0}, common.QuestionKanjiListen},
		{"a kanji meaning variety question is a look-alike", kanji, kanjiOne,
			QuestionCard{State: srs.StateLearning, Facet: common.CardFacetMeaning},
			common.AnswerModeMixed, []int{0}, common.QuestionKanjiLookalike},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.card.Facet == "" {
				tc.card.Facet = common.CardFacetRecognition
			}
			item, _ := tc.pool.Item(tc.item)
			q := NewQuestionBuilder(&fixedRand{ints: tc.rolls}).Build(&QuestionInput{Card: tc.card, Item: item, Pool: tc.pool, Mode: tc.mode})
			if q.QuestionType != string(tc.want) {
				t.Errorf("question type = %s, want %s", q.QuestionType, tc.want)
			}
		})
	}
}

func TestQuestionBuilderContent(t *testing.T) {
	hiragana := loadTestPool(common.LearningTrackHiragana)
	kanji := loadTestPool(common.LearningTrackKanjiN5)
	newCard := QuestionCard{ID: uuid.New(), State: srs.StateNew, Facet: common.CardFacetRecognition}
	tests := []struct {
		name         string
		pool         *Pool
		item         uuid.UUID
		card         QuestionCard
		confused     []uuid.UUID
		rolls        []int
		wantPrompt   string
		wantAnswer   string
		wantAccepted []string
		wantOptions  []string
	}{
		{
			name: "romaji choice lists past confusions and look-alikes first", pool: hiragana, item: kanaA, card: newCard,
			confused: []uuid.UUID{kanaZu}, rolls: []int{0},
			wantPrompt: "あ", wantAnswer: "a", wantOptions: []string{"a", "zu", "o", "i"},
		},
		{
			name: "options never include a kana with the same sound", pool: hiragana, item: kanaJi, card: newCard,
			rolls: []int{1}, wantPrompt: "ji", wantAnswer: "じ", wantOptions: []string{"じ", "ず", "あ", "い"},
		},
		{
			name: "typed kana carries its alternative romaji", pool: hiragana, item: kanaJi,
			card:       QuestionCard{State: srs.StateReview, Stability: 5, Facet: common.CardFacetRecognition},
			wantPrompt: "じ", wantAnswer: "ji", wantAccepted: []string{"zi"},
		},
		{
			name: "meaning choice skips a kanji that shares a meaning", pool: kanji, item: kanjiMiddle,
			card:       QuestionCard{State: srs.StateNew, Facet: common.CardFacetMeaning},
			wantPrompt: "中", wantAnswer: "middle", wantAccepted: []string{"inside"},
			wantOptions: []string{"middle", "one", "two", "day"},
		},
		{
			name: "reading choice accepts every reading", pool: kanji, item: kanjiOne,
			card:       QuestionCard{State: srs.StateNew, Facet: common.CardFacetReading},
			wantPrompt: "一", wantAnswer: "イチ", wantAccepted: []string{"イツ", "ひと", "ひとつ"},
			wantOptions: []string{"イチ", "ニ", "チュウ", "ハン"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item, _ := tc.pool.Item(tc.item)
			q := NewQuestionBuilder(&fixedRand{ints: tc.rolls}).Build(&QuestionInput{
				Card: tc.card, Item: item, Pool: tc.pool, Confused: tc.confused, Mode: common.AnswerModeMixed,
			})
			if q.Prompt != tc.wantPrompt || q.CorrectAnswer != tc.wantAnswer {
				t.Errorf("prompt/answer = %q/%q, want %q/%q", q.Prompt, q.CorrectAnswer, tc.wantPrompt, tc.wantAnswer)
			}
			if !slices.Equal(q.AcceptedAnswers, tc.wantAccepted) && len(q.AcceptedAnswers)+len(tc.wantAccepted) > 0 {
				t.Errorf("accepted = %v, want %v", q.AcceptedAnswers, tc.wantAccepted)
			}
			if !slices.Equal(q.Options, tc.wantOptions) {
				t.Errorf("options = %v, want %v", q.Options, tc.wantOptions)
			}
			if q.Char != item.Char || q.CardID != tc.card.ID {
				t.Errorf("char/card = %s/%s, want %s/%s", q.Char, q.CardID, item.Char, tc.card.ID)
			}
		})
	}
}
