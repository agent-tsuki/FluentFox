package learn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type KanjiNoteRepo interface {
	FindKanjiID(ctx context.Context, char string) (uuid.UUID, error)
	GetKanjiNote(ctx context.Context, userID, kanjiID uuid.UUID) (*KanjiNote, error)
	SaveKanjiNote(ctx context.Context, n *KanjiNote) error
	DeleteKanjiNote(ctx context.Context, userID, kanjiID uuid.UUID) error
}

var _ KanjiNoteRepo = (*Repository)(nil)

type KanjiNoteService struct {
	repo   KanjiNoteRepo
	now    func() time.Time
	logger *zap.Logger
}

func NewKanjiNoteService(repo KanjiNoteRepo, log *zap.Logger) *KanjiNoteService {
	return &KanjiNoteService{repo: repo, now: time.Now, logger: log}
}

// Get returns an empty note when the user has none.
func (s *KanjiNoteService) Get(ctx context.Context, userID uuid.UUID, char string) (KanjiNoteResponse, error) {
	kanjiID, err := s.kanjiID(ctx, char)
	if err != nil {
		return KanjiNoteResponse{}, err
	}
	note, err := s.repo.GetKanjiNote(ctx, userID, kanjiID)
	if err != nil {
		return KanjiNoteResponse{}, fmt.Errorf("get kanji note: %w", err)
	}
	if note == nil {
		return KanjiNoteResponse{}, nil
	}
	return KanjiNoteResponse{Text: note.Text, UpdatedAt: &note.UpdatedAt}, nil
}

// Put saves the note; text that is empty after trimming deletes it.
func (s *KanjiNoteService) Put(ctx context.Context, userID uuid.UUID, char string, req *KanjiNoteRequest) (KanjiNoteResponse, error) {
	text := strings.TrimSpace(req.Text)
	if utf8.RuneCountInString(text) > common.KANJI_NOTE_MAX_CHARS {
		return KanjiNoteResponse{}, exceptions.ErrKanjiNoteTooLong
	}
	kanjiID, err := s.kanjiID(ctx, char)
	if err != nil {
		return KanjiNoteResponse{}, err
	}
	if text == "" {
		if err := s.repo.DeleteKanjiNote(ctx, userID, kanjiID); err != nil {
			return KanjiNoteResponse{}, fmt.Errorf("delete kanji note: %w", err)
		}
		return KanjiNoteResponse{}, nil
	}
	note := KanjiNote{UserID: userID, KanjiID: kanjiID, Text: text, UpdatedAt: s.now().UTC()}
	if err := s.repo.SaveKanjiNote(ctx, &note); err != nil {
		return KanjiNoteResponse{}, fmt.Errorf("save kanji note: %w", err)
	}
	return KanjiNoteResponse{Text: note.Text, UpdatedAt: &note.UpdatedAt}, nil
}

func (s *KanjiNoteService) kanjiID(ctx context.Context, char string) (uuid.UUID, error) {
	id, err := s.repo.FindKanjiID(ctx, char)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, exceptions.ErrKanjiNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("kanji note: %w", err)
	}
	return id, nil
}
