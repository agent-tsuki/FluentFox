package content

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

// ChapterWriter is what ChapterSyncService needs.
type ChapterWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	UpsertGrammarChapter(ctx context.Context, tx *gorm.DB, chapter *GrammarChapter) (GrammarChapter, error)
	UpsertChapterWord(ctx context.Context, tx *gorm.DB, word *ChapterWord) error
	UpsertGrammarSentence(ctx context.Context, tx *gorm.DB, sentence *GrammarSentence) error
}

var _ ChapterWriter = (*Repository)(nil)

type ChapterSyncService struct {
	repo   ChapterWriter
	logger *zap.Logger
}

func NewChapterSyncService(repo ChapterWriter, log *zap.Logger) *ChapterSyncService {
	return &ChapterSyncService{repo: repo, logger: log}
}

// One transaction per file, so a bad file leaves the database as it was.
func (s *ChapterSyncService) Sync(ctx context.Context, file *ChapterFile) error {
	words, sentences := 0, 0
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		for i := range file.Chapters {
			w, st, err := s.syncChapter(ctx, tx, file.Level, &file.Chapters[i])
			if err != nil {
				return err
			}
			words, sentences = words+w, sentences+st
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sync chapters %s: %w", file.Level, err)
	}
	s.logger.Info("grammar chapters synced", zap.String("level", string(file.Level)),
		zap.Int("chapters", len(file.Chapters)), zap.Int("words", words), zap.Int("sentences", sentences))
	return nil
}

func (s *ChapterSyncService) syncChapter(ctx context.Context, tx *gorm.DB, level common.JLPTLevel, c *ChapterFileEntry) (int, int, error) {
	chapter, err := s.repo.UpsertGrammarChapter(ctx, tx, &GrammarChapter{
		Level: level, Slug: c.Slug, Number: c.Number, Title: c.Title, Position: c.Position,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("chapter %s: %w", c.Slug, err)
	}
	for i, w := range c.Words {
		word := ChapterWord{
			ChapterID: chapter.ID, Position: i + 1, Word: w.Word, Kana: w.Kana, Romaji: w.Romaji, Meaning: w.Meaning, Hint: w.Hint,
		}
		if err := s.repo.UpsertChapterWord(ctx, tx, &word); err != nil {
			return 0, 0, fmt.Errorf("chapter %s word %s: %w", c.Slug, w.Word, err)
		}
	}
	for i := range c.Sentences {
		st := &c.Sentences[i]
		sentence := GrammarSentence{
			ChapterID: chapter.ID, Position: i + 1, Sentence: st.Sentence, Answer: st.Answer,
			Distractors: st.Distractors, AlsoAccept: nonNil(st.AlsoAccept), Romaji: nonNil(st.Romaji),
			Translation: st.Translation, Explanation: st.Explanation,
		}
		if err := s.repo.UpsertGrammarSentence(ctx, tx, &sentence); err != nil {
			return 0, 0, fmt.Errorf("chapter %s sentence %d: %w", c.Slug, i+1, err)
		}
	}
	return len(c.Words), len(c.Sentences), nil
}
