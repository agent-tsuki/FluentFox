package learn

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	if err := r.db.WithContext(ctx).Transaction(fn); err != nil {
		return fmt.Errorf("learn transaction: %w", err)
	}
	return nil
}

// EnsureSettings creates the row with the column defaults on first use, so
// the defaults live in one place: the migration.
func (r *Repository) EnsureSettings(ctx context.Context, userID uuid.UUID) (LearningSettings, error) {
	db := r.db.WithContext(ctx)
	if err := db.Exec("INSERT INTO learning_settings (user_id) VALUES (?) ON CONFLICT (user_id) DO NOTHING", userID).Error; err != nil {
		return LearningSettings{}, fmt.Errorf("create learning settings for %s: %w", userID, err)
	}
	var s LearningSettings
	if err := db.Where("user_id = ?", userID).First(&s).Error; err != nil {
		return LearningSettings{}, fmt.Errorf("get learning settings for %s: %w", userID, err)
	}
	return s, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, s *LearningSettings) error {
	err := r.db.WithContext(ctx).Model(&LearningSettings{}).Where("user_id = ?", s.UserID).
		Select("desired_retention", "new_kana_per_day", "new_kanji_per_day", "max_reviews_per_day", "session_size",
			"lesson_size", "daily_goal_answers", "day_starts_at_hour", "answer_mode", "show_retention", "updated_at").
		Updates(s).Error
	if err != nil {
		return fmt.Errorf("update learning settings for %s: %w", s.UserID, err)
	}
	return nil
}

// users_settings.user_id is text (built by the old AutoMigrate), hence String().
// An empty result means the user has no time zone set.
func (r *Repository) GetTimeZone(ctx context.Context, userID uuid.UUID) (string, error) {
	var zones []sql.NullString
	err := r.db.WithContext(ctx).Table("users_settings").Where("user_id = ?", userID.String()).
		Limit(1).Pluck("current_time_zone", &zones).Error
	if err != nil {
		return "", fmt.Errorf("get time zone for %s: %w", userID, err)
	}
	if len(zones) == 0 || !zones[0].Valid {
		return "", nil
	}
	return zones[0].String, nil
}

func (r *Repository) CountDueCards(ctx context.Context, userID uuid.UUID, before time.Time, track *common.LearningTrack) (int, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&SRSCard{}).Where("user_id = ? AND state <> 0 AND due <= ?", userID, before)
	if err := withTrack(q, track).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("count due cards for %s: %w", userID, err)
	}
	return int(n), nil
}

func (r *Repository) CountLearnedItems(ctx context.Context, userID uuid.UUID, track *common.LearningTrack) (int, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&SRSCard{}).Select("COUNT(DISTINCT (item_type, item_id))").
		Where("user_id = ? AND state <> 0", userID)
	if err := withTrack(q, track).Scan(&n).Error; err != nil {
		return 0, fmt.Errorf("count learned items for %s: %w", userID, err)
	}
	return int(n), nil
}

func (r *Repository) CountItemsIntroducedSince(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType, since time.Time) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&SRSCard{}).Select("COUNT(DISTINCT item_id)").
		Where("user_id = ? AND item_type = ? AND introduced_at >= ?", userID, itemType, since).Scan(&n).Error
	if err != nil {
		return 0, fmt.Errorf("count items introduced for %s: %w", userID, err)
	}
	return int(n), nil
}

func (r *Repository) ListCardKeys(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType) ([]CardKey, error) {
	var keys []CardKey
	err := r.db.WithContext(ctx).Model(&SRSCard{}).Select("item_id, facet").
		Where("user_id = ? AND item_type = ?", userID, itemType).Scan(&keys).Error
	if err != nil {
		return nil, fmt.Errorf("list card keys for %s: %w", userID, err)
	}
	return keys, nil
}

// A missing row reads as an empty day.
func (r *Repository) GetDailyActivity(ctx context.Context, userID uuid.UUID, localDate string) (DailyActivity, error) {
	var rows []DailyActivity
	err := r.db.WithContext(ctx).Where("user_id = ? AND local_date = ?::date", userID, localDate).Limit(1).Find(&rows).Error
	if err != nil {
		return DailyActivity{}, fmt.Errorf("get daily activity for %s on %s: %w", userID, localDate, err)
	}
	if len(rows) == 0 {
		return DailyActivity{}, nil
	}
	return rows[0], nil
}

// Reviews are answers on cards that were already introduced (state_before <> 0).
func (r *Repository) CountReviewAnswersSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&ReviewLog{}).
		Where("user_id = ? AND answered_at >= ? AND state_before <> 0", userID, since).Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("count review answers for %s: %w", userID, err)
	}
	return int(n), nil
}

func (r *Repository) ListConfusions(ctx context.Context, userID uuid.UUID) ([]UserConfusion, error) {
	var rows []UserConfusion
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("times DESC, last_at DESC").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list confusions for %s: %w", userID, err)
	}
	return rows, nil
}

func (r *Repository) CreateSession(ctx context.Context, s *StudySession) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("create %s session for %s: %w", s.Kind, s.UserID, err)
	}
	return nil
}

func (r *Repository) GetSession(ctx context.Context, userID, sessionID uuid.UUID) (StudySession, error) {
	var s StudySession
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&s).Error; err != nil {
		return StudySession{}, fmt.Errorf("get session %s: %w", sessionID, err)
	}
	return s, nil
}

func (r *Repository) ListDueCards(ctx context.Context, userID uuid.UUID, now time.Time, track *common.LearningTrack) ([]SRSCard, error) {
	var cards []SRSCard
	q := r.db.WithContext(ctx).Where("user_id = ? AND state <> 0 AND due <= ?", userID, now)
	if err := withTrack(q, track).Order("due").Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("list due cards for %s: %w", userID, err)
	}
	return cards, nil
}

func (r *Repository) ListCardsByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]SRSCard, error) {
	var cards []SRSCard
	if len(ids) == 0 {
		return cards, nil
	}
	if err := r.db.WithContext(ctx).Where("user_id = ? AND id IN ?", userID, ids).Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("list cards for %s: %w", userID, err)
	}
	return cards, nil
}

// Answers with idle or implausible times say nothing about the user's usual
// speed, so they stay out of the baseline.
func (r *Repository) ListBaselineSamples(ctx context.Context, userID uuid.UUID, qt common.QuestionType, limit int) ([]BaselineSample, error) {
	var rows []BaselineSample
	err := r.db.WithContext(ctx).Model(&ReviewLog{}).Select("active_ms, first_input_ms, answer_text").
		Where("user_id = ? AND question_type = ? AND is_correct AND grade_reason NOT IN ?", userID, qt, baselineExcludedReasons).
		Order("answered_at DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list baseline samples for %s %s: %w", userID, qt, err)
	}
	return rows, nil
}

var baselineExcludedReasons = []string{string(common.GradeReasonIdle), string(common.GradeReasonTooFast)}

func (r *Repository) ListItemChars(ctx context.Context, itemType common.SRSItemType, ids []uuid.UUID) ([]ItemChar, error) {
	var rows []ItemChar
	if len(ids) == 0 {
		return rows, nil
	}
	q := r.db.WithContext(ctx)
	if itemType == common.SRSItemKana {
		q = q.Table("kana").Select("id, character AS char")
	} else {
		q = q.Table("kanji").Select("id, char")
	}
	if err := q.Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list %s chars: %w", itemType, err)
	}
	return rows, nil
}

func withTrack(q *gorm.DB, track *common.LearningTrack) *gorm.DB {
	if track == nil {
		return q
	}
	if script, ok := common.LEARNING_TRACK_KANA_SCRIPT[*track]; ok {
		return q.Where("item_type = ? AND item_id IN (SELECT id FROM kana WHERE script = ?)", common.SRSItemKana, script)
	}
	return q.Where("item_type = ? AND item_id IN (SELECT id FROM kanji WHERE level = ?)",
		common.SRSItemKanji, common.LEARNING_TRACK_KANJI_LEVEL[*track])
}
