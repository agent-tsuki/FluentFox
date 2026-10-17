package learn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

type CompleteSessionService struct {
	repo   CompleteRepo
	now    func() time.Time
	logger *zap.Logger
}

func NewCompleteSessionService(repo CompleteRepo, log *zap.Logger) *CompleteSessionService {
	return &CompleteSessionService{repo: repo, now: time.Now, logger: log}
}

// Complete closes the session and returns its result screen. Completing a
// session twice returns the same result, so a client retry is harmless.
func (s *CompleteSessionService) Complete(ctx context.Context, userID, sessionID uuid.UUID) (SessionResult, error) {
	var session StudySession
	var logs []ReviewLog
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var txErr error
		session, logs, txErr = s.close(ctx, tx, userID, sessionID)
		return txErr
	})
	if err != nil {
		return SessionResult{}, fmt.Errorf("complete session: %w", err)
	}
	cards, err := s.cardResults(ctx, userID, logs)
	if err != nil {
		return SessionResult{}, fmt.Errorf("complete session: %w", err)
	}
	return SessionResult{
		Session:      toSessionDTO(&session),
		AnswerCount:  session.AnswerCount,
		CorrectCount: session.CorrectCount,
		ActiveMs:     session.ActiveMs,
		Counts:       countLabels(cards),
		Cards:        cards,
	}, nil
}

// Totals come from the review log rather than running counters, so they are
// right however the answers were batched or resent.
func (s *CompleteSessionService) close(ctx context.Context, tx *gorm.DB, userID, sessionID uuid.UUID) (StudySession, []ReviewLog, error) {
	session, err := s.repo.LockSession(ctx, tx, sessionID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && session.UserID != userID) {
		return StudySession{}, nil, exceptions.ErrStudySessionNotFound
	}
	if err != nil {
		return StudySession{}, nil, fmt.Errorf("close session: %w", err)
	}
	logs, err := s.repo.ListSessionLogs(ctx, tx, sessionID)
	if err != nil {
		return StudySession{}, nil, fmt.Errorf("close session: %w", err)
	}
	if session.CompletedAt != nil {
		return session, logs, nil
	}
	session.AnswerCount, session.CorrectCount, session.ActiveMs = len(logs), 0, 0
	for i := range logs {
		if logs[i].IsCorrect {
			session.CorrectCount++
		}
		session.ActiveMs += int64(logs[i].ActiveMs)
	}
	completedAt := s.now().UTC()
	session.CompletedAt = &completedAt
	if err := s.repo.CompleteSession(ctx, tx, &session); err != nil {
		return StudySession{}, nil, fmt.Errorf("close session: %w", err)
	}
	return session, logs, nil
}

// Each card is shown with its first graded answer of the session, the one
// that moved its schedule; later repeats of a learning card are practice.
func firstGradedLogs(logs []ReviewLog) []*ReviewLog {
	var order []*ReviewLog
	index := map[uuid.UUID]int{}
	for i := range logs {
		log := &logs[i]
		at, seen := index[log.CardID]
		switch {
		case !seen:
			index[log.CardID] = len(order)
			order = append(order, log)
		case order[at].Rating == nil && log.Rating != nil:
			order[at] = log
		}
	}
	return order
}

func (s *CompleteSessionService) cardResults(ctx context.Context, userID uuid.UUID, logs []ReviewLog) ([]SessionCardResult, error) {
	firsts := firstGradedLogs(logs)
	ids := make([]uuid.UUID, 0, len(firsts))
	types := make([]common.QuestionType, 0, len(firsts))
	for _, l := range firsts {
		ids = append(ids, l.CardID)
		types = append(types, l.QuestionType)
	}
	cards, err := s.repo.ListCardsByIDs(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("card results: %w", err)
	}
	chars, err := s.itemChars(ctx, cards)
	if err != nil {
		return nil, err
	}
	baselines, err := loadBaselines(ctx, s.repo, userID, types)
	if err != nil {
		return nil, fmt.Errorf("card results: %w", err)
	}
	byID := make(map[uuid.UUID]*SRSCard, len(cards))
	for i := range cards {
		byID[cards[i].ID] = &cards[i]
	}
	out := make([]SessionCardResult, 0, len(firsts))
	for _, l := range firsts {
		card, ok := byID[l.CardID]
		if !ok {
			continue
		}
		out = append(out, cardResult(l, card, chars[card.ItemID], baselines[l.QuestionType]))
	}
	return out, nil
}

func cardResult(log *ReviewLog, card *SRSCard, char string, b Baseline) SessionCardResult {
	tiles := 0
	if log.Options != nil {
		tiles = len(*log.Options)
	}
	res := SessionCardResult{
		CardID:   card.ID,
		ItemType: string(card.ItemType),
		ItemID:   card.ItemID,
		Facet:    string(card.Facet),
		Char:     char,
		Label:    string(labelOf(log.Rating, log.IsCorrect)),
		Reason:   string(log.GradeReason),
		ActiveMs: log.ActiveMs,
		UsualMs:  int(usualTime(log.QuestionType, answerChars(log.AnswerText), tiles, b) / time.Millisecond),
	}
	if card.State != srs.StateNew {
		due := card.Due
		res.NextDue = &due
	}
	return res
}

func (s *CompleteSessionService) itemChars(ctx context.Context, cards []SRSCard) (map[uuid.UUID]string, error) {
	ids := map[common.SRSItemType][]uuid.UUID{}
	for i := range cards {
		ids[cards[i].ItemType] = append(ids[cards[i].ItemType], cards[i].ItemID)
	}
	out := make(map[uuid.UUID]string, len(cards))
	for itemType, itemIDs := range ids {
		rows, err := s.repo.ListItemChars(ctx, itemType, itemIDs)
		if err != nil {
			return nil, fmt.Errorf("item chars: %w", err)
		}
		for _, r := range rows {
			out[r.ID] = r.Char
		}
	}
	return out, nil
}

func countLabels(cards []SessionCardResult) GradeCounts {
	var c GradeCounts
	for i := range cards {
		switch common.GradeLabel(cards[i].Label) {
		case common.GradeLabelStrong:
			c.Strong++
		case common.GradeLabelGood:
			c.Good++
		case common.GradeLabelShaky:
			c.Shaky++
		default:
			c.Missed++
		}
	}
	return c
}
