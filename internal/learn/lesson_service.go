package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

type StartLessonService struct {
	repo      LessonRepo
	catalog   *Catalog
	questions *QuestionBuilder
	now       func() time.Time
	logger    *zap.Logger
}

func NewStartLessonService(repo LessonRepo, catalog *Catalog, questions *QuestionBuilder, log *zap.Logger) *StartLessonService {
	return &StartLessonService{repo: repo, catalog: catalog, questions: questions, now: time.Now, logger: log}
}

// Start picks the next items in path order that still miss a card. Cards are
// only created when their first check answer arrives, so an abandoned lesson
// leaves nothing behind and its items come up again. The session has no
// questions when the daily limit is used up or the track is done.
func (s *StartLessonService) Start(ctx context.Context, userID uuid.UUID, req *StartLessonRequest) (SessionStart, error) {
	track, err := parseTrack(req.Track)
	if err != nil {
		return SessionStart{}, err
	}
	now := s.now()
	l, err := loadLearner(ctx, s.repo, userID, now)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start lesson: %w", err)
	}
	itemType := common.LEARNING_TRACK_ITEM_TYPE[track]
	introduced, err := s.repo.CountItemsIntroducedSince(ctx, userID, itemType, l.Day.Start)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start lesson: %w", err)
	}
	keys, err := s.repo.ListCardKeys(ctx, userID, itemType)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start lesson: %w", err)
	}
	pool, err := s.catalog.Load(ctx, track)
	if err != nil {
		return SessionStart{}, fmt.Errorf("start lesson: %w", err)
	}

	limit := min(l.Settings.LessonSize, max(l.newPerDay(itemType)-introduced, 0))
	pending := pendingItems(pool, keys)
	items := pending[:min(limit, len(pending))]

	session := newStudySession(userID, common.SessionKindLesson, &track, now)
	if err := s.repo.CreateSession(ctx, &session); err != nil {
		return SessionStart{}, fmt.Errorf("start lesson: %w", err)
	}
	return SessionStart{
		Session:   toSessionDTO(&session),
		Questions: s.checkQuestions(&l, pool, items, keys, l.Settings.AnswerMode),
		Items:     toLessonItems(items),
	}, nil
}

func (s *StartLessonService) checkQuestions(l *learner, pool *Pool, items []*Item, keys []CardKey, mode common.AnswerMode) []Question {
	have := facetsByItem(keys)
	questions := make([]Question, 0, len(items))
	for _, item := range items {
		for _, facet := range missingFacets(item, have[item.ID]) {
			card := QuestionCard{ID: cardID(l.UserID, item.Type, item.ID, facet), State: srs.StateNew, Facet: facet}
			questions = append(questions, s.questions.Build(&QuestionInput{Card: card, Item: item, Pool: pool, Mode: mode}))
		}
	}
	return questions
}

func toLessonItems(items []*Item) []LessonItem {
	out := make([]LessonItem, 0, len(items))
	for _, item := range items {
		out = append(out, LessonItem{
			ItemType:   string(item.Type),
			ItemID:     item.ID,
			Char:       item.Char,
			Romaji:     item.Romaji,
			Meanings:   item.Meanings,
			KeyReading: item.KeyReading,
		})
	}
	return out
}

func newStudySession(userID uuid.UUID, kind common.SessionKind, track *common.LearningTrack, now time.Time) StudySession {
	return StudySession{ID: uuid.New(), UserID: userID, Kind: kind, Track: track, StartedAt: now.UTC()}
}

func toSessionDTO(s *StudySession) StudySessionDTO {
	dto := StudySessionDTO{ID: s.ID, Kind: string(s.Kind), Scope: s.Scope, StartedAt: s.StartedAt, CompletedAt: s.CompletedAt}
	if s.Track != nil {
		track := string(*s.Track)
		dto.Track = &track
	}
	return dto
}
