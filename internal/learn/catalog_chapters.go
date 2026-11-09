package learn

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
)

// ChapterReader is what the chapter tracks need; *content.Repository
// satisfies it.
type ChapterReader interface {
	ListGrammarChapters(ctx context.Context, level common.JLPTLevel) ([]content.GrammarChapter, error)
	ListChapterWords(ctx context.Context, level common.JLPTLevel) ([]content.ChapterWord, error)
	ListGrammarSentences(ctx context.Context, level common.JLPTLevel) ([]content.GrammarSentence, error)
}

// loadChapters builds a chapter track: the level's chapters are its units,
// and its items are the chapters' words (vocab track) or practice sentences
// (grammar track), in chapter order.
func (c *Catalog) loadChapters(ctx context.Context, track common.LearningTrack, level common.JLPTLevel) (*Pool, error) {
	chapters, err := c.content.ListGrammarChapters(ctx, level)
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}
	units := make([]Unit, 0, len(chapters))
	for i := range chapters {
		units = append(units, Unit{
			Type: common.LEARNING_TRACK_UNIT_TYPE[track], ID: chapters[i].ID, Slug: chapters[i].Slug, Title: chapters[i].Title,
		})
	}
	var items []Item
	if common.LEARNING_TRACK_ITEM_TYPE[track] == common.SRSItemVocab {
		items, err = c.chapterWordItems(ctx, level)
	} else {
		items, err = c.sentenceItems(ctx, level)
	}
	if err != nil {
		return nil, fmt.Errorf("load %s pool: %w", track, err)
	}
	return newPool(track, units, items), nil
}

func (c *Catalog) chapterWordItems(ctx context.Context, level common.JLPTLevel) ([]Item, error) {
	words, err := c.content.ListChapterWords(ctx, level)
	if err != nil {
		return nil, fmt.Errorf("chapter words: %w", err)
	}
	items := make([]Item, 0, len(words))
	for i := range words {
		w := &words[i]
		items = append(items, Item{
			Type: common.SRSItemVocab, ID: w.ID, Char: w.Word, UnitID: w.ChapterID, Romaji: w.Romaji,
			Meanings: wordMeanings(w.Meaning), Readings: []string{w.Kana}, KeyReading: w.Kana,
		})
	}
	return items, nil
}

func (c *Catalog) sentenceItems(ctx context.Context, level common.JLPTLevel) ([]Item, error) {
	sentences, err := c.content.ListGrammarSentences(ctx, level)
	if err != nil {
		return nil, fmt.Errorf("grammar sentences: %w", err)
	}
	items := make([]Item, 0, len(sentences))
	for i := range sentences {
		s := &sentences[i]
		items = append(items, Item{
			Type: common.SRSItemGrammarSentence, ID: s.ID, Char: s.Sentence, UnitID: s.ChapterID,
			Sentence: &SentenceItem{
				Answer: s.Answer, Distractors: s.Distractors, AlsoAccept: s.AlsoAccept, Romaji: s.Romaji,
				Translation: s.Translation, Explanation: s.Explanation,
			},
		})
	}
	return items, nil
}

var meaningSeparators = regexp.MustCompile(`\s*[,/;]\s*`)

// wordMeanings keeps a word's meaning as written first (what questions show),
// then each part on its own, so "I, me" also accepts "me" as a typed answer.
func wordMeanings(meaning string) []string {
	out := []string{meaning}
	parts := meaningSeparators.Split(meaning, -1)
	if len(parts) < 2 {
		return out
	}
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return uniqueStrings(out)
}

func (p *Pool) chapterItems(chapterID uuid.UUID) []*Item {
	return p.ItemsWhere(func(u Unit) bool { return u.ID == chapterID })
}
