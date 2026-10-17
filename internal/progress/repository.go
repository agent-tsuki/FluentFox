package progress

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
	"github.com/fluentfox/api/internal/learn"
)

// *learn.Repository satisfies every repo interface here, and
// *learn.Catalog satisfies PoolLoader.

type PoolLoader interface {
	Pools(ctx context.Context, track *common.LearningTrack) (learn.Pools, error)
}

type CardStatsRepo interface {
	ListUserCards(ctx context.Context, userID uuid.UUID) ([]learn.SRSCard, error)
	ListCardAnswerStats(ctx context.Context, userID uuid.UUID, f learn.AnswerStatsFilter) ([]learn.CardAnswerStats, error)
}

type GoalDaysRepo interface {
	ListGoalMetDates(ctx context.Context, userID uuid.UUID) ([]time.Time, error)
}

type OverviewRepo interface {
	learn.LearnerRepo
	CardStatsRepo
	GoalDaysRepo
	GetDailyActivity(ctx context.Context, userID uuid.UUID, localDate string) (learn.DailyActivity, error)
	CountDueCards(ctx context.Context, userID uuid.UUID, before time.Time, track *common.LearningTrack) (int, error)
	CountItemsIntroducedSince(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType, since time.Time) (int, error)
	CountReviewAnswersSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
}

type ForecastRepo interface {
	learn.LearnerRepo
	ListDueCards(ctx context.Context, userID uuid.UUID, now time.Time, track *common.LearningTrack) ([]learn.SRSCard, error)
}

type CalendarRepo interface {
	learn.LearnerRepo
	GoalDaysRepo
	ListDailyActivity(ctx context.Context, userID uuid.UUID, from, to string) ([]learn.DailyActivity, error)
}

type WeakItemsRepo interface {
	CardStatsRepo
	EnsureSettings(ctx context.Context, userID uuid.UUID) (learn.LearningSettings, error)
	ListConfusions(ctx context.Context, userID uuid.UUID) ([]learn.UserConfusion, error)
}

type ConfusionRepo interface {
	ListConfusions(ctx context.Context, userID uuid.UUID) ([]learn.UserConfusion, error)
}

type ItemHistoryRepo interface {
	EnsureSettings(ctx context.Context, userID uuid.UUID) (learn.LearningSettings, error)
	ListItemCards(ctx context.Context, userID uuid.UUID, itemType common.SRSItemType, itemID uuid.UUID) ([]learn.SRSCard, error)
	ListRecentAnswers(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID, limit int) ([]learn.ReviewLog, error)
	ListCardAnswerStats(ctx context.Context, userID uuid.UUID, f learn.AnswerStatsFilter) ([]learn.CardAnswerStats, error)
}

// KanaReader resolves kana slugs, which the learning pools do not carry;
// *content.Repository satisfies it.
type KanaReader interface {
	ListKana(ctx context.Context, script common.KanaScript) ([]content.Kana, error)
}

type DashboardRepo interface {
	learn.LearnerRepo
	CardStatsRepo
	GoalDaysRepo
	ListDailyActivity(ctx context.Context, userID uuid.UUID, from, to string) ([]learn.DailyActivity, error)
	SumActiveMs(ctx context.Context, userID uuid.UUID) (int64, error)
	CountReviewAnswersSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	GetTargetLevel(ctx context.Context, userID uuid.UUID) (string, error)
	ListRecentSessions(ctx context.Context, userID uuid.UUID, limit int) ([]learn.StudySession, error)
}

var (
	_ OverviewRepo    = (*learn.Repository)(nil)
	_ ForecastRepo    = (*learn.Repository)(nil)
	_ CalendarRepo    = (*learn.Repository)(nil)
	_ WeakItemsRepo   = (*learn.Repository)(nil)
	_ ConfusionRepo   = (*learn.Repository)(nil)
	_ ItemHistoryRepo = (*learn.Repository)(nil)
	_ DashboardRepo   = (*learn.Repository)(nil)
	_ KanaReader      = (*content.Repository)(nil)
	_ PoolLoader      = (*learn.Catalog)(nil)
)
