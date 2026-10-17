package progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

func newItemHistoryService(repo *fakeRepo) *ItemHistoryService {
	svc := NewItemHistoryService(repo, &fakePools{pools: testPools()}, fakeKana{}, zap.NewNop())
	svc.now = func() time.Time { return testNow }
	return svc
}

func TestItemHistoryFindsTheItem(t *testing.T) {
	tests := []struct {
		name     string
		in       ItemHistoryInput
		wantChar string
		wantErr  error
	}{
		{"kana slug in one script", ItemHistoryInput{Type: "kana", ID: "ga"}, "が", nil},
		{"kana slug in both scripts needs the track", ItemHistoryInput{Type: "kana", ID: "a"}, "", exceptions.ErrProgressItemAmbiguous},
		{"kana slug with the track", ItemHistoryInput{Type: "kana", ID: "a", Track: "katakana"}, "ア", nil},
		{"kana character", ItemHistoryInput{Type: "kana", ID: "い"}, "い", nil},
		{"kanji character", ItemHistoryInput{Type: "kanji", ID: "二"}, "二", nil},
		{"item id", ItemHistoryInput{Type: "kanji", ID: kanji1.String()}, "一", nil},
		{"kana asked as kanji is not found", ItemHistoryInput{Type: "kanji", ID: "あ"}, "", exceptions.ErrProgressItemNotFound},
		{"unknown slug", ItemHistoryInput{Type: "kana", ID: "zz"}, "", exceptions.ErrProgressItemNotFound},
		{"bad track", ItemHistoryInput{Type: "kana", ID: "a", Track: "n4"}, "", exceptions.ErrLearningTrackInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newItemHistoryService(newFakeRepo()).Get(context.Background(), testUserID, &tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got.Char != tt.wantChar {
				t.Errorf("char = %q, want %q", got.Char, tt.wantChar)
			}
		})
	}
}

func TestItemHistoryCardsAndAnswers(t *testing.T) {
	repo := newFakeRepo()
	meaning := repo.card(common.SRSItemKanji, kanji1, common.CardFacetMeaning, srs.StateReview, 30)
	reading := repo.card(common.SRSItemKanji, kanji1, common.CardFacetReading, srs.StateNew, 0)
	repo.card(common.SRSItemKanji, kanji2, common.CardFacetMeaning, srs.StateReview, 30)
	good := srs.RatingGood
	repo.answers = []learn.ReviewLog{
		{CardID: meaning, QuestionType: common.QuestionKanjiMeaningMCQ, IsCorrect: true, Rating: &good, GradeReason: common.GradeReasonMCQCap, ActiveMs: 2100},
		{CardID: reading, QuestionType: common.QuestionKanjiReadingMCQ, IsCorrect: false, GradeReason: common.GradeReasonAnalyticsOnly, ActiveMs: 4000},
		{CardID: uuid.New(), QuestionType: common.QuestionKanjiMeaningMCQ, IsCorrect: true},
	}
	repo.stats = []learn.CardAnswerStats{
		stat(meaning, common.QuestionKanjiMeaningMCQ, 4, 3, 2, 2, 3, 6300),
		stat(reading, common.QuestionKanjiReadingMCQ, 1, 0, 0, 0, 0, 0),
	}
	got, err := newItemHistoryService(repo).Get(context.Background(), testUserID, &ItemHistoryInput{Type: "kanji", ID: "一"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Level != string(common.MasteryNew) || got.Track != "kanji_n5" || len(got.Cards) != 2 {
		t.Fatalf("item = level %s track %s cards %d; want new, kanji_n5, 2", got.Level, got.Track, len(got.Cards))
	}
	cardTests := []struct {
		name       string
		card       ItemCardProgress
		wantLevel  string
		wantRecall bool
	}{
		{"scheduled card has a recall chance and due date", got.Cards[0], "mastered", true},
		{"unscheduled card has neither", got.Cards[1], "new", false},
	}
	for _, tt := range cardTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.card.Level != tt.wantLevel || (tt.card.Recall != nil) != tt.wantRecall || (tt.card.NextDue != nil) != tt.wantRecall {
				t.Errorf("card = %+v", tt.card)
			}
		})
	}
	if len(got.Answers) != 2 || got.Answers[0].Label == nil || *got.Answers[0].Label != "good" || got.Answers[1].Label != nil {
		t.Errorf("answers = %+v; want two, the rated one labelled good and the analytics one unlabelled", got.Answers)
	}
	wantStats := ItemStats{Answers: 5, Correct: 3, Reviews: 2}
	if got.Stats.Answers != wantStats.Answers || got.Stats.Correct != wantStats.Correct || got.Stats.Reviews != wantStats.Reviews ||
		*got.Stats.Accuracy != 0.6 || *got.Stats.Retention != 1 || *got.Stats.AvgAnswerMs != 2100 {
		t.Errorf("stats = %+v", got.Stats)
	}
}

func TestItemHistoryWithoutCards(t *testing.T) {
	got, err := newItemHistoryService(newFakeRepo()).Get(context.Background(), testUserID, &ItemHistoryInput{Type: "kana", ID: "い"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Level != "new" || len(got.Cards) != 0 || len(got.Answers) != 0 || got.Stats.Accuracy != nil {
		t.Errorf("history = %+v", got)
	}
}
