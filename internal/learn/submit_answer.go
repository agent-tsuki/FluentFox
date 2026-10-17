package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

// answer grades one new answer, schedules its card when the answer counts,
// and stores the review log.
func (s *SubmitAnswersService) answer(ctx context.Context, tx *gorm.DB, b *answerBatch, a *AnswerRequestItem) (ReviewLog, error) {
	card := b.cards[a.CardID]
	qt := common.QuestionType(a.QuestionType)
	if common.QUESTION_TYPE_FACET[qt] != card.Facet {
		return ReviewLog{}, exceptions.ErrQuestionTypeMismatch
	}
	pool, item, ok := b.pools.Find(card.ItemType, card.ItemID)
	if !ok {
		return ReviewLog{}, fmt.Errorf("answer %s: card %s has no content item", a.ClientEventID, card.ID)
	}
	check := checkAnswer(qt, item, pool, a)
	grade := Grade(gradeInput(qt, a, &check), b.baselines[qt])
	log := newReviewLog(b, card, a, &check, &grade)

	wasNew := card.State == srs.StateNew
	if err := s.applyGrade(ctx, tx, b, card, &log, &grade); err != nil {
		return ReviewLog{}, err
	}
	if err := s.repo.InsertReviewLog(ctx, tx, &log); err != nil {
		return ReviewLog{}, fmt.Errorf("answer %s: %w", a.ClientEventID, err)
	}
	if err := s.recordConfusion(ctx, tx, b, item, &check, log.AnsweredAt); err != nil {
		return ReviewLog{}, err
	}
	b.addToDay(&log, wasNew && log.CountsForSRS && card.Facet == common.SRS_ITEM_FACETS[card.ItemType][0])
	return log, nil
}

// Learning steps need every answer; a card in review only learns from its
// first graded answer of the day.
func (s *SubmitAnswersService) applyGrade(ctx context.Context, tx *gorm.DB, b *answerBatch, card *SRSCard, log *ReviewLog, grade *GradeResult) error {
	log.CountsForSRS = grade.Rated && (card.State != srs.StateReview || !b.rated[card.ID])
	if grade.Rated {
		rating := grade.Rating
		log.Rating = &rating
		b.rated[card.ID] = true
	}
	if log.CountsForSRS {
		return s.schedule(ctx, tx, b, card, log)
	}
	if card.State != srs.StateNew {
		due := card.Due
		log.DueAfter = &due
		log.ScheduledDays = card.ScheduledDays
	}
	return nil
}

func (s *SubmitAnswersService) schedule(ctx context.Context, tx *gorm.DB, b *answerBatch, card *SRSCard, log *ReviewLog) error {
	before := toSRSCard(card)
	next, info, err := b.sched.Next(&before, log.AnsweredAt, *log.Rating)
	if err != nil {
		return fmt.Errorf("schedule card %s: %w", card.ID, err)
	}
	log.RetrievabilityBefore = info.RetrievabilityBefore
	log.ElapsedDays = info.ElapsedDays
	if card.State != srs.StateNew {
		stability, difficulty := card.Stability, card.Difficulty
		log.StabilityBefore, log.DifficultyBefore = &stability, &difficulty
	}

	card.State, card.Due, card.Stability, card.Difficulty = next.State, next.Due, next.Stability, next.Difficulty
	card.Reps, card.Lapses, card.LearningStep, card.ScheduledDays = next.Reps, next.Lapses, next.LearningStep, next.ScheduledDays
	card.LastReviewAt = next.LastReview
	// A leech stays scheduled; the flag only makes the client show extra help.
	if card.Lapses >= common.LEECH_LAPSES {
		card.IsLeech = true
	}
	stability, difficulty, due := card.Stability, card.Difficulty, card.Due
	log.StabilityAfter, log.DifficultyAfter, log.DueAfter = &stability, &difficulty, &due
	log.ScheduledDays = card.ScheduledDays
	if err := s.repo.SaveCard(ctx, tx, card); err != nil {
		return fmt.Errorf("schedule card %s: %w", card.ID, err)
	}
	return nil
}

func (s *SubmitAnswersService) recordConfusion(ctx context.Context, tx *gorm.DB, b *answerBatch, item *Item, check *answerCheck, at time.Time) error {
	if check.Correct || check.Chosen == nil {
		return nil
	}
	err := s.repo.UpsertConfusion(ctx, tx, &UserConfusion{
		UserID: b.learner.UserID, ItemType: item.Type, TargetItemID: item.ID, ChosenItemID: check.Chosen.ID, LastAt: at,
	})
	if err != nil {
		return fmt.Errorf("record confusion: %w", err)
	}
	return nil
}

func (b *answerBatch) addToDay(log *ReviewLog, newItem bool) {
	b.delta.ReviewsDone++
	b.delta.ActiveMs += int64(log.ActiveMs)
	if log.IsCorrect {
		b.delta.CorrectCount++
	}
	if log.Rating != nil && *log.Rating == srs.RatingAgain {
		b.delta.AgainCount++
	}
	if newItem {
		b.delta.NewLearned++
	}
}

func gradeInput(qt common.QuestionType, a *AnswerRequestItem, check *answerCheck) *GradeInput {
	in := &GradeInput{
		QuestionType: qt,
		Correct:      check.Correct,
		Revealed:     a.AnswerRevealed,
		Attempts:     a.Attempts,
		HintsUsed:    a.HintsUsed,
		TypoDistance: check.TypoDistance,
		AudioReplays: a.AudioReplays,
		AnswerChars:  answerChars(a.AnswerText),
		Tiles:        len(a.Options),
		StrokeErrors: derefInt(a.StrokeErrors),
		StrokeCount:  derefInt(a.StrokeCount),
		Duration:     msDuration(a.DurationMs),
		Idle:         msDuration(a.IdleMs),
		Audio:        msDuration(derefInt(a.AudioMs)),
	}
	if a.GuideLevel != nil {
		in.GuideLevel = common.GuideLevel(*a.GuideLevel)
	}
	return in
}

func newReviewLog(b *answerBatch, card *SRSCard, a *AnswerRequestItem, check *answerCheck, grade *GradeResult) ReviewLog {
	log := ReviewLog{
		ID:             uuid.New(),
		UserID:         b.learner.UserID,
		CardID:         card.ID,
		SessionID:      b.session.ID,
		ClientEventID:  a.ClientEventID,
		QuestionType:   common.QuestionType(a.QuestionType),
		AnsweredAt:     answerTime(a, b.session.StartedAt, card.LastReviewAt, b.now),
		ShownAt:        a.ShownAt,
		DurationMs:     a.DurationMs,
		IdleMs:         a.IdleMs,
		FirstInputMs:   a.FirstInputMs,
		Attempts:       a.Attempts,
		AnswerText:     a.AnswerText,
		ChosenOption:   a.ChosenOption,
		HintsUsed:      a.HintsUsed,
		AnswerRevealed: a.AnswerRevealed,
		AudioReplays:   a.AudioReplays,
		AudioMs:        a.AudioMs,
		StrokeErrors:   a.StrokeErrors,
		StrokeCount:    a.StrokeCount,
		IsCorrect:      check.Correct,
		ActiveMs:       int(grade.Active / time.Millisecond),
		GradeReason:    grade.Reason,
		StateBefore:    card.State,
	}
	if a.Options != nil {
		options := database.TextArray(a.Options)
		log.Options = &options
	}
	if a.GuideLevel != nil {
		guide := common.GuideLevel(*a.GuideLevel)
		log.GuideLevel = &guide
	}
	if check.TypoDistance > 0 {
		typo := check.TypoDistance
		log.TypoDistance = &typo
	}
	return log
}

// answerTime uses the client's own answer time because answers arrive in
// batches, and learning steps are minutes apart. It is clamped to the
// session and the server clock so a wrong client clock cannot move a card,
// and never goes before the card's last review.
func answerTime(a *AnswerRequestItem, sessionStart time.Time, lastReview *time.Time, now time.Time) time.Time {
	t := a.ShownAt.Add(msDuration(a.DurationMs))
	if t.Before(sessionStart) {
		t = sessionStart
	}
	if lastReview != nil && t.Before(*lastReview) {
		t = *lastReview
	}
	if t.After(now) {
		t = now
	}
	return t.UTC()
}

// result is built from the stored log, so a resent answer gets the same
// result as the first time (usual_ms is recomputed from the current baseline).
func (b *answerBatch) result(log *ReviewLog) AnswerResult {
	res := AnswerResult{
		ClientEventID: log.ClientEventID,
		Correct:       log.IsCorrect,
		Label:         string(labelOf(log.Rating, log.IsCorrect)),
		Reason:        string(log.GradeReason),
		ActiveMs:      log.ActiveMs,
		NextDue:       log.DueAfter,
		// In short-term mode a learning or relearning step has no whole days.
		Requeue: log.CountsForSRS && log.DueAfter != nil && log.ScheduledDays == 0,
	}
	tiles := 0
	if log.Options != nil {
		tiles = len(*log.Options)
	}
	res.UsualMs = int(usualTime(log.QuestionType, answerChars(log.AnswerText), tiles, b.baselines[log.QuestionType]) / time.Millisecond)
	if card, ok := b.cards[log.CardID]; ok {
		if _, item, found := b.pools.Find(card.ItemType, card.ItemID); found {
			res.CorrectAnswer = itemValue(item, questionAnswerKinds[log.QuestionType])
		}
	}
	return res
}
