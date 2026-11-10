package users

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

// UserTutorial is one learner's state for one in-app tutorial.
type UserTutorial struct {
	UserID      uuid.UUID             `gorm:"column:user_id;type:uuid;primaryKey"`
	TutorialKey string                `gorm:"column:tutorial_key;primaryKey"`
	Status      common.TutorialStatus `gorm:"column:status;type:tutorial_status"`
	Version     int                   `gorm:"column:version"`
	LastStep    *int                  `gorm:"column:last_step"`
	CompletedAt *time.Time            `gorm:"column:completed_at"`
	UpdatedAt   time.Time             `gorm:"column:updated_at"`
}

func (UserTutorial) TableName() string { return "user_tutorial" }

func (r *Repository) ListUserTutorials(ctx context.Context, userID uuid.UUID) ([]UserTutorial, error) {
	var rows []UserTutorial
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list tutorials for %s: %w", userID, err)
	}
	return rows, nil
}

// UpsertUserTutorial writes t and returns the stored row. completed_at is set
// on every completion, so a replayed tour records when it was last finished.
func (r *Repository) UpsertUserTutorial(ctx context.Context, t *UserTutorial) (UserTutorial, error) {
	var saved UserTutorial
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO user_tutorial (user_id, tutorial_key, status, version, last_step, completed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, tutorial_key) DO UPDATE SET
			status       = EXCLUDED.status,
			version      = EXCLUDED.version,
			last_step    = EXCLUDED.last_step,
			completed_at = COALESCE(EXCLUDED.completed_at, user_tutorial.completed_at),
			updated_at   = EXCLUDED.updated_at
		RETURNING user_id, tutorial_key, status, version, last_step, completed_at, updated_at`,
		t.UserID, t.TutorialKey, t.Status, t.Version, t.LastStep, t.CompletedAt, t.UpdatedAt).
		Scan(&saved).Error
	if err != nil {
		return UserTutorial{}, fmt.Errorf("upsert tutorial %s for %s: %w", t.TutorialKey, t.UserID, err)
	}
	return saved, nil
}
