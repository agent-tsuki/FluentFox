package learn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

// CardAnswerStats sums one card's answers of one question type. Reviews are
// scheduled answers on cards in the review state; Passed are those not rated
// Again. Timed answers are the correct ones with a plausible time, as in the
// grading baseline.
type CardAnswerStats struct {
	CardID       uuid.UUID           `gorm:"column:card_id"`
	QuestionType common.QuestionType `gorm:"column:question_type"`
	Answers      int                 `gorm:"column:answers"`
	Correct      int                 `gorm:"column:correct"`
	Reviews      int                 `gorm:"column:reviews"`
	Passed       int                 `gorm:"column:passed"`
	TimedAnswers int                 `gorm:"column:timed_answers"`
	TimedMs      int64               `gorm:"column:timed_ms"`
}

// AnswerStatsFilter limits ListCardAnswerStats; a zero Since and nil CardIDs
// mean every answer of the user.
type AnswerStatsFilter struct {
	Since   time.Time
	CardIDs []uuid.UUID
}

func (r *Repository) ListUserCards(ctx context.Context, userID uuid.UUID) ([]SRSCard, error) {
	var cards []SRSCard
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("list cards for %s: %w", userID, err)
	}
	return cards, nil
}

func (r *Repository) ListItemCards(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType, itemID uuid.UUID) ([]SRSCard, error) {
	var cards []SRSCard
	err := r.db.WithContext(ctx).Where("user_id = ? AND item_type = ? AND item_id = ?", userID, itemType, itemID).
		Order("facet").Find(&cards).Error
	if err != nil {
		return nil, fmt.Errorf("list %s %s cards for %s: %w", itemType, itemID, userID, err)
	}
	return cards, nil
}

func (r *Repository) ListCardAnswerStats(ctx context.Context, userID uuid.UUID, f AnswerStatsFilter) ([]CardAnswerStats, error) {
	var rows []CardAnswerStats
	q := r.db.WithContext(ctx).Model(&ReviewLog{}).Select(`card_id, question_type,
		COUNT(*) AS answers,
		COUNT(*) FILTER (WHERE is_correct) AS correct,
		COUNT(*) FILTER (WHERE counts_for_srs AND state_before = ?) AS reviews,
		COUNT(*) FILTER (WHERE counts_for_srs AND state_before = ? AND rating > ?) AS passed,
		COUNT(*) FILTER (WHERE is_correct AND grade_reason NOT IN ?) AS timed_answers,
		COALESCE(SUM(active_ms) FILTER (WHERE is_correct AND grade_reason NOT IN ?), 0) AS timed_ms`,
		srs.StateReview, srs.StateReview, srs.RatingAgain, baselineExcludedReasons, baselineExcludedReasons).
		Where("user_id = ? AND answered_at >= ?", userID, f.Since)
	if f.CardIDs != nil {
		q = q.Where("card_id IN ?", f.CardIDs)
	}
	if err := q.Group("card_id, question_type").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list answer stats for %s: %w", userID, err)
	}
	return rows, nil
}

// ListRecentAnswers returns the newest answers on the cards, newest first.
func (r *Repository) ListRecentAnswers(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID, limit int) ([]ReviewLog, error) {
	var rows []ReviewLog
	if len(cardIDs) == 0 {
		return rows, nil
	}
	err := r.db.WithContext(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).
		Order("answered_at DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list recent answers for %s: %w", userID, err)
	}
	return rows, nil
}

// ListDailyActivity returns the rows between two local dates, both included,
// in date order. Days without answers have no row.
func (r *Repository) ListDailyActivity(ctx context.Context, userID uuid.UUID, from, to string) ([]DailyActivity, error) {
	var rows []DailyActivity
	err := r.db.WithContext(ctx).Where("user_id = ? AND local_date BETWEEN ?::date AND ?::date", userID, from, to).
		Order("local_date").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list daily activity for %s: %w", userID, err)
	}
	return rows, nil
}

// ListGoalMetDates returns every local date the daily goal was met, oldest
// first, as midnight UTC like DailyActivity.LocalDate.
func (r *Repository) ListGoalMetDates(ctx context.Context, userID uuid.UUID) ([]time.Time, error) {
	var dates []time.Time
	err := r.db.WithContext(ctx).Model(&DailyActivity{}).Where("user_id = ? AND goal_met", userID).
		Order("local_date").Pluck("local_date", &dates).Error
	if err != nil {
		return nil, fmt.Errorf("list goal days for %s: %w", userID, err)
	}
	return dates, nil
}

func (r *Repository) SumActiveMs(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&DailyActivity{}).Select("COALESCE(SUM(active_ms), 0)").
		Where("user_id = ?", userID).Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("sum active time for %s: %w", userID, err)
	}
	return total, nil
}

// Sessions without answers (opened and left) are skipped.
func (r *Repository) ListRecentSessions(ctx context.Context, userID uuid.UUID, limit int) ([]StudySession, error) {
	var rows []StudySession
	err := r.db.WithContext(ctx).Where("user_id = ? AND answer_count > 0", userID).
		Order("started_at DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list recent sessions for %s: %w", userID, err)
	}
	return rows, nil
}

// users_profile.user_id is text (built by the old AutoMigrate), hence
// String(). An empty result means no target level is set.
func (r *Repository) GetTargetLevel(ctx context.Context, userID uuid.UUID) (string, error) {
	var levels []sql.NullString
	err := r.db.WithContext(ctx).Table("users_profile").Where("user_id = ?", userID.String()).
		Limit(1).Pluck("target_level", &levels).Error
	if err != nil {
		return "", fmt.Errorf("get target level for %s: %w", userID, err)
	}
	if len(levels) == 0 || !levels[0].Valid {
		return "", nil
	}
	return levels[0].String, nil
}

// ListTestResults returns every test the user took, oldest first.
func (r *Repository) ListTestResults(ctx context.Context, userID uuid.UUID, track *common.LearningTrack) ([]TestResult, error) {
	var rows []TestResult
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if track != nil {
		q = q.Where("track = ?", *track)
	}
	if err := q.Order("taken_at").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list test results for %s: %w", userID, err)
	}
	return rows, nil
}
