package learn

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

type SubmitAnswersService struct {
	repo    SubmitRepo
	catalog *Catalog
	now     func() time.Time
	logger  *zap.Logger
}

func NewSubmitAnswersService(repo SubmitRepo, catalog *Catalog, log *zap.Logger) *SubmitAnswersService {
	return &SubmitAnswersService{repo: repo, catalog: catalog, now: time.Now, logger: log}
}

// answerBatch is the state shared by the answers of one request. cards and
// rated change as answers are applied, so a card answered twice in a batch
// is scheduled from its updated state.
type answerBatch struct {
	learner   *learner
	session   StudySession
	sched     *srs.Scheduler
	pools     Pools
	baselines map[common.QuestionType]Baseline
	now       time.Time
	cards     map[uuid.UUID]*SRSCard
	rated     map[uuid.UUID]bool
	delta     DailyActivity
}

func (s *SubmitAnswersService) Submit(ctx context.Context, userID, sessionID uuid.UUID, req *SubmitAnswersRequest) (SubmitAnswersResponse, error) {
	if err := validateAnswers(req.Answers); err != nil {
		return SubmitAnswersResponse{}, err
	}
	b, err := s.prepare(ctx, userID, sessionID, req.Answers)
	if err != nil {
		return SubmitAnswersResponse{}, err
	}
	var results []AnswerResult
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var applyErr error
		results, applyErr = s.apply(ctx, tx, b, req.Answers)
		return applyErr
	})
	if err != nil {
		return SubmitAnswersResponse{}, fmt.Errorf("submit answers: %w", err)
	}
	return SubmitAnswersResponse{Results: results}, nil
}

func validateAnswers(answers []AnswerRequestItem) error {
	if len(answers) == 0 || len(answers) > common.MAX_ANSWERS_PER_BATCH {
		return exceptions.ErrValidation
	}
	for i := range answers {
		if _, ok := common.QUESTION_TYPE_FACET[common.QuestionType(answers[i].QuestionType)]; !ok {
			return exceptions.ErrQuestionTypeMismatch
		}
	}
	return nil
}

// prepare does the reads that need no lock before the transaction starts,
// so the rows stay locked for as short a time as possible.
func (s *SubmitAnswersService) prepare(ctx context.Context, userID, sessionID uuid.UUID, answers []AnswerRequestItem) (*answerBatch, error) {
	session, err := s.repo.GetSession(ctx, userID, sessionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.ErrStudySessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("submit answers: %w", err)
	}
	now := s.now()
	l, err := loadLearner(ctx, s.repo, userID, now)
	if err != nil {
		return nil, fmt.Errorf("submit answers: %w", err)
	}
	sched, err := newScheduler(&l.Settings)
	if err != nil {
		return nil, fmt.Errorf("submit answers: %w", err)
	}
	pools, err := s.catalog.Pools(ctx, session.Track)
	if err != nil {
		return nil, fmt.Errorf("submit answers: %w", err)
	}
	types := make([]common.QuestionType, 0, len(answers))
	for i := range answers {
		types = append(types, common.QuestionType(answers[i].QuestionType))
	}
	baselines, err := loadBaselines(ctx, s.repo, userID, types)
	if err != nil {
		return nil, fmt.Errorf("submit answers: %w", err)
	}
	return &answerBatch{
		learner: &l, session: session, sched: sched, pools: pools, baselines: baselines, now: now,
		rated: map[uuid.UUID]bool{}, delta: DailyActivity{UserID: userID},
	}, nil
}

// apply runs inside the transaction. The session row is locked first, then
// the cards in id order, so concurrent batches of one session queue up
// instead of deadlocking.
func (s *SubmitAnswersService) apply(ctx context.Context, tx *gorm.DB, b *answerBatch, answers []AnswerRequestItem) ([]AnswerResult, error) {
	session, err := s.repo.LockSession(ctx, tx, b.session.ID)
	if err != nil {
		return nil, fmt.Errorf("apply answers: %w", err)
	}
	if session.CompletedAt != nil {
		return nil, exceptions.ErrStudySessionCompleted
	}
	b.session = session
	done, err := s.storedLogs(ctx, tx, b, answers)
	if err != nil {
		return nil, err
	}
	if err := s.lockCards(ctx, tx, b, answers); err != nil {
		return nil, err
	}
	dueBefore, err := s.repo.CountDueCardsTx(ctx, tx, b.learner.UserID, b.now)
	if err != nil {
		return nil, fmt.Errorf("apply answers: %w", err)
	}

	results := make([]AnswerResult, 0, len(answers))
	for i := range answers {
		a := &answers[i]
		log, ok := done[a.ClientEventID]
		if !ok {
			if log, err = s.answer(ctx, tx, b, a); err != nil {
				return nil, err
			}
			done[a.ClientEventID] = log
		}
		results = append(results, b.result(&log))
	}
	if err := s.recordActivity(ctx, tx, b, dueBefore); err != nil {
		return nil, err
	}
	return results, nil
}

// A resent answer returns the result stored the first time.
func (s *SubmitAnswersService) storedLogs(ctx context.Context, tx *gorm.DB, b *answerBatch, answers []AnswerRequestItem) (map[uuid.UUID]ReviewLog, error) {
	ids := make([]uuid.UUID, 0, len(answers))
	for i := range answers {
		ids = append(ids, answers[i].ClientEventID)
	}
	logs, err := s.repo.FindReviewLogs(ctx, tx, b.learner.UserID, ids)
	if err != nil {
		return nil, fmt.Errorf("apply answers: %w", err)
	}
	out := make(map[uuid.UUID]ReviewLog, len(logs))
	for i := range logs {
		out[logs[i].ClientEventID] = logs[i]
	}
	return out, nil
}

func (s *SubmitAnswersService) lockCards(ctx context.Context, tx *gorm.DB, b *answerBatch, answers []AnswerRequestItem) error {
	ids := make([]uuid.UUID, 0, len(answers))
	for i := range answers {
		ids = append(ids, answers[i].CardID)
	}
	slices.SortFunc(ids, func(x, y uuid.UUID) int { return slices.Compare(x[:], y[:]) })
	ids = slices.Compact(ids)

	if err := s.createMissingCards(ctx, tx, b, ids); err != nil {
		return err
	}
	locked, err := s.repo.LockCards(ctx, tx, b.learner.UserID, ids)
	if err != nil {
		return fmt.Errorf("lock cards: %w", err)
	}
	if len(locked) != len(ids) {
		return exceptions.ErrSRSCardNotFound
	}
	b.cards = make(map[uuid.UUID]*SRSCard, len(locked))
	for i := range locked {
		b.cards[locked[i].ID] = &locked[i]
	}
	rated, err := s.repo.ListCardsRatedSince(ctx, tx, ids, b.learner.Day.Start)
	if err != nil {
		return fmt.Errorf("lock cards: %w", err)
	}
	for _, id := range rated {
		b.rated[id] = true
	}
	return nil
}

func (s *SubmitAnswersService) createMissingCards(ctx context.Context, tx *gorm.DB, b *answerBatch, ids []uuid.UUID) error {
	existing, err := s.repo.ListExistingCardIDs(ctx, tx, b.learner.UserID, ids)
	if err != nil {
		return fmt.Errorf("create missing cards: %w", err)
	}
	missing := slices.DeleteFunc(slices.Clone(ids), func(id uuid.UUID) bool { return slices.Contains(existing, id) })
	if len(missing) == 0 {
		return nil
	}
	created, err := b.newLessonCards(missing)
	if err != nil {
		return err
	}
	if err := s.repo.CreateCards(ctx, tx, created); err != nil {
		return fmt.Errorf("create missing cards: %w", err)
	}
	return nil
}

// newLessonCards creates the cards a lesson handed out ids for. Only items of
// the session's track qualify, since card ids are derived from the item.
func (b *answerBatch) newLessonCards(ids []uuid.UUID) ([]SRSCard, error) {
	if b.session.Kind != common.SessionKindLesson || len(b.pools) == 0 {
		return nil, exceptions.ErrSRSCardNotFound
	}
	userID := b.learner.UserID
	byID := make(map[uuid.UUID]SRSCard)
	pool := b.pools[0]
	for i := range pool.Items {
		item := &pool.Items[i]
		for _, facet := range common.SRS_ITEM_FACETS[item.Type] {
			id := cardID(userID, item.Type, item.ID, facet)
			byID[id] = SRSCard{
				ID: id, UserID: userID, ItemType: item.Type, ItemID: item.ID, Facet: facet,
				State: srs.StateNew, Due: b.now, IntroducedAt: b.now,
			}
		}
	}
	out := make([]SRSCard, 0, len(ids))
	for _, id := range ids {
		card, ok := byID[id]
		if !ok {
			return nil, exceptions.ErrSRSCardNotFound
		}
		out = append(out, card)
	}
	return out, nil
}

// recordActivity adds the batch to the day's totals. The goal is met at
// daily_goal_answers answers, or when nothing is left due after at least
// GOAL_CLEARED_MIN_ANSWERS answers; once met it stays met.
func (s *SubmitAnswersService) recordActivity(ctx context.Context, tx *gorm.DB, b *answerBatch, dueBefore int) error {
	if b.delta.ReviewsDone == 0 {
		return nil
	}
	b.delta.DueAtDayStart = dueBefore
	day, err := s.repo.AddDailyActivity(ctx, tx, &b.delta, b.learner.Day.LocalDate)
	if err != nil {
		return fmt.Errorf("record activity: %w", err)
	}
	if day.GoalMet {
		return nil
	}
	met := day.ReviewsDone >= b.learner.Settings.DailyGoalAnswers
	if !met && day.ReviewsDone >= common.GOAL_CLEARED_MIN_ANSWERS {
		due, err := s.repo.CountDueCardsTx(ctx, tx, b.learner.UserID, b.now)
		if err != nil {
			return fmt.Errorf("record activity: %w", err)
		}
		met = due == 0
	}
	if !met {
		return nil
	}
	if err := s.repo.MarkGoalMet(ctx, tx, b.learner.UserID, b.learner.Day.LocalDate); err != nil {
		return fmt.Errorf("record activity: %w", err)
	}
	return nil
}
