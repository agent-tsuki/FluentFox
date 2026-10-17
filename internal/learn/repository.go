package learn

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
)

// LearnerRepo gives the user's settings and time zone, which every service
// needs to find the user's study day. *Repository satisfies it, as it does
// every interface below.
type LearnerRepo interface {
	EnsureSettings(ctx context.Context, userID uuid.UUID) (LearningSettings, error)
	GetTimeZone(ctx context.Context, userID uuid.UUID) (string, error)
}

type SettingsRepo interface {
	EnsureSettings(ctx context.Context, userID uuid.UUID) (LearningSettings, error)
	UpdateSettings(ctx context.Context, s *LearningSettings) error
}

type SummaryRepo interface {
	LearnerRepo
	CountDueCards(ctx context.Context, userID uuid.UUID, before time.Time, track *common.LearningTrack) (int, error)
	CountLearnedItems(ctx context.Context, userID uuid.UUID, track *common.LearningTrack) (int, error)
	CountItemsIntroducedSince(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType, since time.Time) (int, error)
	ListCardKeys(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType) ([]CardKey, error)
	GetDailyActivity(ctx context.Context, userID uuid.UUID, localDate string) (DailyActivity, error)
}

type LessonRepo interface {
	LearnerRepo
	CountItemsIntroducedSince(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType, since time.Time) (int, error)
	ListCardKeys(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType) ([]CardKey, error)
	CreateSession(ctx context.Context, s *StudySession) error
}

type ReviewRepo interface {
	LearnerRepo
	CountReviewAnswersSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	ListDueCards(ctx context.Context, userID uuid.UUID, now time.Time, track *common.LearningTrack) ([]SRSCard, error)
	ListConfusions(ctx context.Context, userID uuid.UUID) ([]UserConfusion, error)
	CreateSession(ctx context.Context, s *StudySession) error
}

type BaselineRepo interface {
	ListBaselineSamples(ctx context.Context, userID uuid.UUID, qt common.QuestionType, limit int) ([]BaselineSample, error)
}

type SubmitRepo interface {
	LearnerRepo
	BaselineRepo
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	GetSession(ctx context.Context, userID, sessionID uuid.UUID) (StudySession, error)
	LockSession(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) (StudySession, error)
	FindReviewLogs(ctx context.Context, tx *gorm.DB, userID uuid.UUID, eventIDs []uuid.UUID) ([]ReviewLog, error)
	ListExistingCardIDs(ctx context.Context, tx *gorm.DB, userID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error)
	CreateCards(ctx context.Context, tx *gorm.DB, cards []SRSCard) error
	LockCards(ctx context.Context, tx *gorm.DB, userID uuid.UUID, ids []uuid.UUID) ([]SRSCard, error)
	ListCardsRatedSince(ctx context.Context, tx *gorm.DB, cardIDs []uuid.UUID, since time.Time) ([]uuid.UUID, error)
	SaveCard(ctx context.Context, tx *gorm.DB, card *SRSCard) error
	InsertReviewLog(ctx context.Context, tx *gorm.DB, log *ReviewLog) error
	UpsertConfusion(ctx context.Context, tx *gorm.DB, c *UserConfusion) error
	CountDueCardsTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, before time.Time) (int, error)
	AddDailyActivity(ctx context.Context, tx *gorm.DB, delta *DailyActivity, localDate string) (DailyActivity, error)
	MarkGoalMet(ctx context.Context, tx *gorm.DB, userID uuid.UUID, localDate string) error
}

type CompleteRepo interface {
	BaselineRepo
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	LockSession(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) (StudySession, error)
	ListSessionLogs(ctx context.Context, tx *gorm.DB, sessionID uuid.UUID) ([]ReviewLog, error)
	CompleteSession(ctx context.Context, tx *gorm.DB, s *StudySession) error
	ListCardsByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]SRSCard, error)
	ListItemChars(ctx context.Context, itemType common.SRSItemType, ids []uuid.UUID) ([]ItemChar, error)
}

var (
	_ SettingsRepo  = (*Repository)(nil)
	_ SummaryRepo   = (*Repository)(nil)
	_ LessonRepo    = (*Repository)(nil)
	_ ReviewRepo    = (*Repository)(nil)
	_ SubmitRepo    = (*Repository)(nil)
	_ CompleteRepo  = (*Repository)(nil)
	_ ContentReader = (*content.Repository)(nil)
)
