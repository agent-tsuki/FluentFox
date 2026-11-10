package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

const (
	exerciseGapMark     = "{}"
	trueFalseLines      = 2
	exerciseImagePrefix = "/exercises/"
	lessonImagePrefix   = "/images/grammar/"
	minPictureOptions   = 4
	minOrderWords       = 2
	wrongChoices        = 3
)

var (
	errExerciseChapter = errors.New("no such grammar chapter (sync the chapters first)")
	errTrueFalse       = errors.New("true/false needs two lines, a statement and an explanation")
	errTransform       = errors.New("transform needs a source, a task, an answer, an explanation and one {} in its sentence")
	errWordSentence    = errors.New("word sentence needs an answer, a translation and one {} in its sentence")
	errPicture         = errors.New("picture needs an /exercises/ or /images/grammar/ image, alt, translation, one {} in its sentence and four different options with the answer")
	errOrder           = errors.New("order needs at least two words and a translation")
	errListen          = errors.New("listen needs text, a translation, a voice and three other meanings")
	errReply           = errors.New("reply needs a prompt, an answer, an explanation and three wrong replies")
	errReading         = errors.New("reading needs lines, a question, an answer and three wrong answers")
)

// ExerciseWriter is what ExerciseSyncService needs.
type ExerciseWriter interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	ChapterIDBySlug(ctx context.Context, tx *gorm.DB, level common.JLPTLevel, slug string) (uuid.UUID, error)
	UpsertChapterExercise(ctx context.Context, tx *gorm.DB, e *ChapterExercise) error
	TrimChapterExercises(ctx context.Context, tx *gorm.DB, chapterID uuid.UUID, keep int) error
}

var _ ExerciseWriter = (*Repository)(nil)

// ExerciseSyncService loads content/exercises/*.json after the chapters.
type ExerciseSyncService struct {
	repo   ExerciseWriter
	logger *zap.Logger
}

func NewExerciseSyncService(repo ExerciseWriter, log *zap.Logger) *ExerciseSyncService {
	return &ExerciseSyncService{repo: repo, logger: log}
}

// Sync checks the whole file before writing, in one transaction.
func (s *ExerciseSyncService) Sync(ctx context.Context, file *ExerciseFile) error {
	if err := CheckExerciseFile(file); err != nil {
		return fmt.Errorf("exercises %s: %w", file.Level, err)
	}
	items := 0
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		for i := range file.Chapters {
			n, err := s.syncChapter(ctx, tx, file.Level, &file.Chapters[i])
			if err != nil {
				return fmt.Errorf("chapter %s: %w", file.Chapters[i].Slug, err)
			}
			items += n
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sync exercises %s: %w", file.Level, err)
	}
	s.logger.Info("chapter exercises synced", zap.String("level", string(file.Level)), zap.Int("items", items))
	return nil
}

func (s *ExerciseSyncService) syncChapter(ctx context.Context, tx *gorm.DB, level common.JLPTLevel, c *ExerciseFileChapter) (int, error) {
	chapterID, err := s.repo.ChapterIDBySlug(ctx, tx, level, c.Slug)
	if err != nil {
		return 0, fmt.Errorf("find chapter: %w", err)
	}
	if chapterID == uuid.Nil {
		return 0, errExerciseChapter
	}
	rows, err := exerciseRows(c)
	if err != nil {
		return 0, err
	}
	for i := range rows {
		rows[i].ChapterID, rows[i].Position = chapterID, i+1
		if err := s.repo.UpsertChapterExercise(ctx, tx, &rows[i]); err != nil {
			return 0, fmt.Errorf("item %d: %w", i+1, err)
		}
	}
	if err := s.repo.TrimChapterExercises(ctx, tx, chapterID, len(rows)); err != nil {
		return 0, fmt.Errorf("trim: %w", err)
	}
	return len(rows), nil
}

func exerciseRows(c *ExerciseFileChapter) ([]ChapterExercise, error) {
	rows := make([]ChapterExercise, 0, len(c.TrueFalse)+len(c.Transforms)+len(c.Pictures)+len(c.Words))
	add := func(kind ChapterExerciseKind, item any) error {
		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("encode %s item: %w", kind, err)
		}
		rows = append(rows, ChapterExercise{Kind: kind, Data: data})
		return nil
	}
	for i := range c.TrueFalse {
		if err := add(ChapterExerciseTrueFalse, &c.TrueFalse[i]); err != nil {
			return nil, err
		}
	}
	for i := range c.Transforms {
		if err := add(ChapterExerciseTransform, &c.Transforms[i]); err != nil {
			return nil, err
		}
	}
	for i := range c.Pictures {
		if err := add(ChapterExercisePicture, &c.Pictures[i]); err != nil {
			return nil, err
		}
	}
	for i := range c.Words {
		if err := add(ChapterExerciseWord, &c.Words[i]); err != nil {
			return nil, err
		}
	}
	for i := range c.Orders {
		if err := add(ChapterExerciseOrder, &c.Orders[i]); err != nil {
			return nil, err
		}
	}
	for i := range c.Listens {
		if err := add(ChapterExerciseListen, &c.Listens[i]); err != nil {
			return nil, err
		}
	}
	for i := range c.Replies {
		if err := add(ChapterExerciseReply, &c.Replies[i]); err != nil {
			return nil, err
		}
	}
	for i := range c.Readings {
		if err := add(ChapterExerciseReading, &c.Readings[i]); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// CheckExerciseFile reports the first problem in an exercises file.
func CheckExerciseFile(file *ExerciseFile) error {
	for i := range file.Chapters {
		if err := checkExerciseChapter(&file.Chapters[i]); err != nil {
			return fmt.Errorf("chapter %s: %w", file.Chapters[i].Slug, err)
		}
	}
	return nil
}

func checkExerciseChapter(c *ExerciseFileChapter) error {
	for i := range c.TrueFalse {
		if err := checkTrueFalse(&c.TrueFalse[i]); err != nil {
			return fmt.Errorf("true/false %d: %w", i+1, err)
		}
	}
	for i := range c.Transforms {
		if err := checkTransform(&c.Transforms[i]); err != nil {
			return fmt.Errorf("transform %d: %w", i+1, err)
		}
	}
	for i := range c.Pictures {
		if err := checkPicture(&c.Pictures[i]); err != nil {
			return fmt.Errorf("picture %d: %w", i+1, err)
		}
	}
	for i := range c.Words {
		w := &c.Words[i]
		if blank(w.Answer, w.Translation) || strings.Count(w.Sentence, exerciseGapMark) != 1 {
			return fmt.Errorf("word sentence %d: %w", i+1, errWordSentence)
		}
	}
	return checkInteractiveExercises(c)
}

func checkInteractiveExercises(c *ExerciseFileChapter) error {
	for i := range c.Orders {
		o := &c.Orders[i]
		if len(o.Words) < minOrderWords || blank(o.Words...) || blank(o.Translation) {
			return fmt.Errorf("order %d: %w", i+1, errOrder)
		}
	}
	for i := range c.Listens {
		l := &c.Listens[i]
		if blank(l.Text, l.Translation, l.Voice) || len(l.Wrong) != wrongChoices || blank(l.Wrong...) {
			return fmt.Errorf("listen %d: %w", i+1, errListen)
		}
	}
	for i := range c.Replies {
		r := &c.Replies[i]
		if blank(r.Prompt, r.Answer, r.Explanation) || len(r.Wrong) != wrongChoices || blank(r.Wrong...) {
			return fmt.Errorf("reply %d: %w", i+1, errReply)
		}
	}
	for i := range c.Readings {
		r := &c.Readings[i]
		if len(r.Lines) == 0 || blank(r.Lines...) || blank(r.Question, r.Answer) || len(r.Wrong) != wrongChoices || blank(r.Wrong...) {
			return fmt.Errorf("reading %d: %w", i+1, errReading)
		}
	}
	return nil
}

func blank(values ...string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return true
		}
	}
	return false
}

func checkTrueFalse(t *TrueFalseItem) error {
	if len(t.Lines) != trueFalseLines || blank(t.Lines...) || blank(t.Statement, t.Explanation) {
		return errTrueFalse
	}
	return nil
}

func checkTransform(t *TransformItem) error {
	if blank(t.Source, t.Task, t.Answer, t.Explanation) || strings.Count(t.Sentence, exerciseGapMark) != 1 {
		return errTransform
	}
	return nil
}

func checkPicture(p *PictureItem) error {
	if !(strings.HasPrefix(p.Image, exerciseImagePrefix) || strings.HasPrefix(p.Image, lessonImagePrefix)) || blank(p.Alt, p.Answer, p.Translation) || strings.Count(p.Sentence, exerciseGapMark) != 1 {
		return errPicture
	}
	seen := map[string]bool{}
	for _, o := range p.Options {
		seen[o] = true
	}
	if len(seen) < minPictureOptions || len(seen) != len(p.Options) || !seen[p.Answer] {
		return errPicture
	}
	return nil
}
