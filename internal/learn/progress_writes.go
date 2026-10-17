package learn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/fluentfox/api/internal/common"
)

func (r *Repository) ListTrackCards(ctx context.Context, userID uuid.UUID, track common.LearningTrack) ([]SRSCard, error) {
	var cards []SRSCard
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if err := withTrack(q, &track).Order("id").Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("list %s cards for %s: %w", track, userID, err)
	}
	return cards, nil
}

// FindTestResult returns nil when the session has no result yet.
func (r *Repository) FindTestResult(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) (*TestResult, error) {
	var rows []TestResult
	if err := tx.WithContext(ctx).Where("session_id = ?", sessionID).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("find test result of session %s: %w", sessionID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (r *Repository) InsertTestResult(ctx context.Context, tx *gorm.DB, res *TestResult) error {
	if err := tx.WithContext(ctx).Create(res).Error; err != nil {
		return fmt.Errorf("insert test result of session %s: %w", res.SessionID, err)
	}
	return nil
}

// RecordUnitQuiz keeps the best score, and the first time the unit was
// completed when p.CompletedAt is set.
func (r *Repository) RecordUnitQuiz(ctx context.Context, tx *gorm.DB, p *UnitProgress) error {
	err := tx.WithContext(ctx).Exec(`
		INSERT INTO unit_progress (user_id, unit_type, unit_id, quiz_best_pct, completed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id, unit_type, unit_id) DO UPDATE SET
			quiz_best_pct = GREATEST(unit_progress.quiz_best_pct, EXCLUDED.quiz_best_pct),
			completed_at  = COALESCE(unit_progress.completed_at, EXCLUDED.completed_at)`,
		p.UserID, p.UnitType, p.UnitID, p.QuizBestPct, p.CompletedAt).Error
	if err != nil {
		return fmt.Errorf("record unit quiz for %s: %w", p.UserID, err)
	}
	return nil
}

// UpsertKnownCards creates the cards a passed skip test proved known. A card
// that already exists keeps its schedule, unless it is a new card a test or
// practice created without scheduling it.
func (r *Repository) UpsertKnownCards(ctx context.Context, tx *gorm.DB, cards []SRSCard) error {
	if len(cards) == 0 {
		return nil
	}
	err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "item_type"}, {Name: "item_id"}, {Name: "facet"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"state", "due", "stability", "difficulty", "reps", "lapses", "learning_step",
			"scheduled_days", "last_review_at", "introduced_at", "updated_at",
		}),
		Where: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "srs_card.state = 0"}}},
	}).Create(&cards).Error
	if err != nil {
		return fmt.Errorf("upsert known cards: %w", err)
	}
	return nil
}

var unitStepColumns = map[common.UnitProgressStep]string{
	common.UnitStepWatched:   "watched_at",
	common.UnitStepTraced:    "traced_at",
	common.UnitStepWritten:   "written_at",
	common.UnitStepPractised: "practised_at",
}

var errUnknownUnitStep = errors.New("unknown unit step")

// MarkUnitStep keeps the first time a step was done.
func (r *Repository) MarkUnitStep(ctx context.Context, p *UnitProgress, step common.UnitProgressStep, at time.Time) (UnitProgress, error) {
	col, ok := unitStepColumns[step]
	if !ok {
		return UnitProgress{}, fmt.Errorf("mark unit step %q: %w", step, errUnknownUnitStep)
	}
	var row UnitProgress
	// col comes from unitStepColumns, never from the request.
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO unit_progress (user_id, unit_type, unit_id, `+col+`)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, unit_type, unit_id) DO UPDATE SET
			`+col+` = COALESCE(unit_progress.`+col+`, EXCLUDED.`+col+`)
		RETURNING *`,
		p.UserID, p.UnitType, p.UnitID, at).Scan(&row).Error
	if err != nil {
		return UnitProgress{}, fmt.Errorf("mark unit %s %s for %s: %w", p.UnitID, step, p.UserID, err)
	}
	return row, nil
}

func (r *Repository) ListUnitProgress(ctx context.Context, userID uuid.UUID) ([]UnitProgress, error) {
	var rows []UnitProgress
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list unit progress for %s: %w", userID, err)
	}
	return rows, nil
}

// FindKanjiID returns gorm.ErrRecordNotFound (wrapped) for an unknown kanji.
func (r *Repository) FindKanjiID(ctx context.Context, char string) (uuid.UUID, error) {
	var row ItemChar
	if err := r.db.WithContext(ctx).Table("kanji").Select("id, char").Where("char = ?", char).Take(&row).Error; err != nil {
		return uuid.Nil, fmt.Errorf("find kanji %s: %w", char, err)
	}
	return row.ID, nil
}

// GetKanjiNote returns nil when the user has no note on the kanji.
func (r *Repository) GetKanjiNote(ctx context.Context, userID, kanjiID uuid.UUID) (*KanjiNote, error) {
	var rows []KanjiNote
	if err := r.db.WithContext(ctx).Where("user_id = ? AND kanji_id = ?", userID, kanjiID).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("get kanji note for %s: %w", userID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (r *Repository) SaveKanjiNote(ctx context.Context, n *KanjiNote) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "kanji_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"text", "updated_at"}),
	}).Create(n).Error
	if err != nil {
		return fmt.Errorf("save kanji note for %s: %w", n.UserID, err)
	}
	return nil
}

func (r *Repository) DeleteKanjiNote(ctx context.Context, userID, kanjiID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("user_id = ? AND kanji_id = ?", userID, kanjiID).Delete(&KanjiNote{}).Error; err != nil {
		return fmt.Errorf("delete kanji note for %s: %w", userID, err)
	}
	return nil
}
