package content

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
)

// ListeningWriter is what ListeningSyncService needs.
type ListeningWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	ChapterIDBySlug(ctx context.Context, tx *gorm.DB, level common.JLPTLevel, slug string) (uuid.UUID, error)
	UpsertChapterPassage(ctx context.Context, tx *gorm.DB, p *ChapterPassage) (uuid.UUID, error)
	UpsertPassageLine(ctx context.Context, tx *gorm.DB, l *ChapterPassageLine) error
	TrimPassages(ctx context.Context, tx *gorm.DB, chapterID uuid.UUID, keep int) error
	TrimPassageLines(ctx context.Context, tx *gorm.DB, passageID uuid.UUID, keep int) error
}

var _ ListeningWriter = (*Repository)(nil)

var errListeningChapter = errors.New("no such grammar chapter (sync the chapters first)")

// ListeningSyncService loads content/listening/*.json: each chapter's
// dialogues and passages. It runs after the chapters' own sync.
type ListeningSyncService struct {
	repo   ListeningWriter
	logger *zap.Logger
}

func NewListeningSyncService(repo ListeningWriter, log *zap.Logger) *ListeningSyncService {
	return &ListeningSyncService{repo: repo, logger: log}
}

// Sync checks the whole file before writing, in one transaction.
func (s *ListeningSyncService) Sync(ctx context.Context, file *ListeningFile) error {
	if err := CheckListeningFile(file); err != nil {
		return fmt.Errorf("listening %s: %w", file.Level, err)
	}
	passages, lines := 0, 0
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		for i := range file.Chapters {
			c := &file.Chapters[i]
			n, err := s.syncChapter(ctx, tx, file.Level, c)
			if err != nil {
				return fmt.Errorf("chapter %s: %w", c.Slug, err)
			}
			passages, lines = passages+len(c.Passages), lines+n
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sync listening %s: %w", file.Level, err)
	}
	s.logger.Info("chapter listening synced", zap.String("level", string(file.Level)),
		zap.Int("chapters", len(file.Chapters)), zap.Int("passages", passages), zap.Int("lines", lines))
	return nil
}

// CheckListeningFile reports the first problem in a listening file.
func CheckListeningFile(file *ListeningFile) error {
	for i := range file.Chapters {
		c := &file.Chapters[i]
		for j := range c.Passages {
			if err := checkPassage(&c.Passages[j]); err != nil {
				return fmt.Errorf("chapter %s passage %d: %w", c.Slug, j+1, err)
			}
		}
	}
	return nil
}

func (s *ListeningSyncService) syncChapter(ctx context.Context, tx *gorm.DB, level common.JLPTLevel, c *ListeningFileChapter) (int, error) {
	chapterID, err := s.repo.ChapterIDBySlug(ctx, tx, level, c.Slug)
	if err != nil {
		return 0, err //nolint:wrapcheck // already wrapped by the repository
	}
	if chapterID == uuid.Nil {
		return 0, errListeningChapter
	}
	lines := 0
	for i := range c.Passages {
		p := &c.Passages[i]
		names, voices := make([]string, len(p.Speakers)), make([]string, len(p.Speakers))
		for j, sp := range p.Speakers {
			names[j], voices[j] = sp.Name, sp.Voice
		}
		passageID, err := s.repo.UpsertChapterPassage(ctx, tx, &ChapterPassage{
			ChapterID: chapterID, Position: i + 1, Kind: p.Kind, Title: p.Title,
			Speakers: database.TextArray(names), SpeakerVoices: database.TextArray(voices),
		})
		if err != nil {
			return 0, err //nolint:wrapcheck // already wrapped by the repository
		}
		for j := range p.Lines {
			l := &p.Lines[j]
			line := ChapterPassageLine{PassageID: passageID, Position: j + 1, Speaker: l.Speaker, Text: l.Text, Translation: l.Translation, Say: l.Say}
			if err := s.repo.UpsertPassageLine(ctx, tx, &line); err != nil {
				return 0, err //nolint:wrapcheck // already wrapped by the repository
			}
		}
		if err := s.repo.TrimPassageLines(ctx, tx, passageID, len(p.Lines)); err != nil {
			return 0, err //nolint:wrapcheck // already wrapped by the repository
		}
		lines += len(p.Lines)
	}
	if err := s.repo.TrimPassages(ctx, tx, chapterID, len(c.Passages)); err != nil {
		return 0, err //nolint:wrapcheck // already wrapped by the repository
	}
	return lines, nil
}
