package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) LockSession(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) (StudySession, error) {
	var s StudySession
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", sessionID).First(&s).Error
	if err != nil {
		return StudySession{}, fmt.Errorf("lock session %s: %w", sessionID, err)
	}
	return s, nil
}

func (r *Repository) FindReviewLogs(ctx context.Context, tx *gorm.DB, userID uuid.UUID, eventIDs []uuid.UUID) ([]ReviewLog, error) {
	var logs []ReviewLog
	if len(eventIDs) == 0 {
		return logs, nil
	}
	err := tx.WithContext(ctx).Where("user_id = ? AND client_event_id IN ?", userID, eventIDs).Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("find review logs for %s: %w", userID, err)
	}
	return logs, nil
}

func (r *Repository) ListExistingCardIDs(ctx context.Context, tx *gorm.DB, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	var found []uuid.UUID
	if len(ids) == 0 {
		return found, nil
	}
	err := tx.WithContext(ctx).Model(&SRSCard{}).Where("user_id = ? AND id IN ?", userID, ids).Pluck("id", &found).Error
	if err != nil {
		return nil, fmt.Errorf("list card ids for %s: %w", userID, err)
	}
	return found, nil
}

// Two batches may create the same card at once; the unique key makes the
// second insert a no-op and the caller then locks the stored row.
func (r *Repository) CreateCards(ctx context.Context, tx *gorm.DB, cards []SRSCard) error {
	if len(cards) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&cards).Error; err != nil {
		return fmt.Errorf("create cards: %w", err)
	}
	return nil
}

// Rows are locked in id order so two batches touching the same cards cannot
// deadlock.
func (r *Repository) LockCards(ctx context.Context, tx *gorm.DB, userID uuid.UUID, ids []uuid.UUID) ([]SRSCard, error) {
	var cards []SRSCard
	if len(ids) == 0 {
		return cards, nil
	}
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("user_id = ? AND id IN ?", userID, ids).Order("id").Find(&cards).Error
	if err != nil {
		return nil, fmt.Errorf("lock cards for %s: %w", userID, err)
	}
	return cards, nil
}

func (r *Repository) SaveCard(ctx context.Context, tx *gorm.DB, card *SRSCard) error {
	if err := tx.WithContext(ctx).Save(card).Error; err != nil {
		return fmt.Errorf("save card %s: %w", card.ID, err)
	}
	return nil
}

func (r *Repository) InsertReviewLog(ctx context.Context, tx *gorm.DB, log *ReviewLog) error {
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("insert review log %s: %w", log.ClientEventID, err)
	}
	return nil
}

func (r *Repository) ListCardsRatedSince(ctx context.Context, tx *gorm.DB, cardIDs []uuid.UUID, since time.Time) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if len(cardIDs) == 0 {
		return ids, nil
	}
	err := tx.WithContext(ctx).Model(&ReviewLog{}).Distinct("card_id").
		Where("card_id IN ? AND answered_at >= ? AND rating IS NOT NULL", cardIDs, since).Pluck("card_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("list cards rated since %s: %w", since, err)
	}
	return ids, nil
}

func (r *Repository) UpsertConfusion(ctx context.Context, tx *gorm.DB, c *UserConfusion) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO user_confusion (user_id, item_type, target_item_id, chosen_item_id, times, last_at)
		VALUES (?, ?, ?, ?, 1, ?)
		ON CONFLICT (user_id, item_type, target_item_id, chosen_item_id)
		DO UPDATE SET times = user_confusion.times + 1, last_at = EXCLUDED.last_at`,
		c.UserID, c.ItemType, c.TargetItemID, c.ChosenItemID, c.LastAt).Error
	if err != nil {
		return fmt.Errorf("upsert confusion for %s: %w", c.UserID, err)
	}
	return nil
}

// AddDailyActivity adds the counters in delta to the day's row and returns
// the new totals. DueAtDayStart is only written when the row is created.
func (r *Repository) AddDailyActivity(ctx context.Context, tx *gorm.DB, delta *DailyActivity, localDate string) (DailyActivity, error) {
	var row DailyActivity
	err := tx.WithContext(ctx).Raw(`
		INSERT INTO daily_activity
			(user_id, local_date, reviews_done, new_learned, correct_count, again_count, active_ms, due_at_day_start)
		VALUES (?, ?::date, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, local_date) DO UPDATE SET
			reviews_done  = daily_activity.reviews_done + EXCLUDED.reviews_done,
			new_learned   = daily_activity.new_learned + EXCLUDED.new_learned,
			correct_count = daily_activity.correct_count + EXCLUDED.correct_count,
			again_count   = daily_activity.again_count + EXCLUDED.again_count,
			active_ms     = daily_activity.active_ms + EXCLUDED.active_ms
		RETURNING *`,
		delta.UserID, localDate, delta.ReviewsDone, delta.NewLearned, delta.CorrectCount,
		delta.AgainCount, delta.ActiveMs, delta.DueAtDayStart).Scan(&row).Error
	if err != nil {
		return DailyActivity{}, fmt.Errorf("add daily activity for %s on %s: %w", delta.UserID, localDate, err)
	}
	return row, nil
}

func (r *Repository) MarkGoalMet(ctx context.Context, tx *gorm.DB, userID uuid.UUID, localDate string) error {
	err := tx.WithContext(ctx).Model(&DailyActivity{}).
		Where("user_id = ? AND local_date = ?::date", userID, localDate).Update("goal_met", true).Error
	if err != nil {
		return fmt.Errorf("mark goal met for %s on %s: %w", userID, localDate, err)
	}
	return nil
}

func (r *Repository) CountDueCardsTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, before time.Time) (int, error) {
	var n int64
	err := tx.WithContext(ctx).Model(&SRSCard{}).Where("user_id = ? AND state <> 0 AND due <= ?", userID, before).Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("count due cards for %s: %w", userID, err)
	}
	return int(n), nil
}

func (r *Repository) ListSessionLogs(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) ([]ReviewLog, error) {
	var logs []ReviewLog
	if err := tx.WithContext(ctx).Where("session_id = ?", sessionID).Order("answered_at, id").Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("list logs of session %s: %w", sessionID, err)
	}
	return logs, nil
}

func (r *Repository) CompleteSession(ctx context.Context, tx *gorm.DB, s *StudySession) error {
	err := tx.WithContext(ctx).Model(&StudySession{}).Where("id = ?", s.ID).
		Select("completed_at", "answer_count", "correct_count", "active_ms").Updates(s).Error
	if err != nil {
		return fmt.Errorf("complete session %s: %w", s.ID, err)
	}
	return nil
}
