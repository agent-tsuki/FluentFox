package content

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type fakeChapters struct {
	chapters  []GrammarChapter
	words     []ChapterWord
	sentences []GrammarSentence
}

func (f *fakeChapters) ListGrammarChapters(_ context.Context, level common.JLPTLevel) ([]GrammarChapter, error) {
	if level != common.JLPTLevelN5 {
		return nil, nil
	}
	return f.chapters, nil
}

func (f *fakeChapters) ListChapterWords(context.Context, common.JLPTLevel) ([]ChapterWord, error) {
	return f.words, nil
}

func (f *fakeChapters) ListGrammarSentences(context.Context, common.JLPTLevel) ([]GrammarSentence, error) {
	return f.sentences, nil
}

func TestChapterContentService(t *testing.T) {
	one, two := uuid.New(), uuid.New()
	hint := "Has 馬 (horse)."
	repo := &fakeChapters{
		chapters: []GrammarChapter{{ID: one, Slug: "ch1", Number: 1, Title: "Identity"}, {ID: two, Slug: "ch2", Number: 2, Title: "Places"}},
		words: []ChapterWord{
			{ChapterID: one, Word: "私", Kana: "わたし", Romaji: "watashi", Meaning: "I, me"},
			{ChapterID: two, Word: "駅", Kana: "えき", Romaji: "eki", Meaning: "station", Hint: &hint},
		},
		sentences: []GrammarSentence{{ChapterID: two, Sentence: "駅{}行きます。", Answer: "へ", Translation: "I go to the station."}},
	}
	tests := []struct {
		name          string
		level, slug   string
		wantErr       error
		wantWords     int
		wantSentences []string
	}{
		{"only the chapter's own words", "n5", "ch1", nil, 1, nil},
		{"sentences are read in full", "n5", "ch2", nil, 1, []string{"駅へ行きます。"}},
		{"an unknown chapter is not found", "n5", "ch9", exceptions.ErrGrammarChapterNotFound, 0, nil},
		{"an unknown level is not found", "n4", "ch1", exceptions.ErrGrammarChapterNotFound, 0, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewChapterContentService(repo, zap.NewNop()).Get(context.Background(), tc.level, tc.slug)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if len(got.Words) != tc.wantWords {
				t.Fatalf("words = %d, want %d", len(got.Words), tc.wantWords)
			}
			for i, want := range tc.wantSentences {
				if got.Sentences[i].Text != want {
					t.Fatalf("sentence %d = %q, want %q", i, got.Sentences[i].Text, want)
				}
			}
		})
	}
}
