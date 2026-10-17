package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
)

type OverviewService struct {
	repo   OverviewRepo
	pools  PoolLoader
	now    func() time.Time
	logger *zap.Logger
}

func NewOverviewService(repo OverviewRepo, pools PoolLoader, log *zap.Logger) *OverviewService {
	return &OverviewService{repo: repo, pools: pools, now: time.Now, logger: log}
}

func (s *OverviewService) Get(ctx context.Context, userID uuid.UUID) (ProgressOverview, error) {
	clock, err := loadClock(ctx, s.repo, userID, s.now())
	if err != nil {
		return ProgressOverview{}, fmt.Errorf("progress overview: %w", err)
	}
	x, stats, err := loadIndexAndStats(ctx, s.repo, s.pools, userID, clock.Now.Add(-common.RETENTION_WINDOW))
	if err != nil {
		return ProgressOverview{}, fmt.Errorf("progress overview: %w", err)
	}
	today, err := s.today(ctx, &clock)
	if err != nil {
		return ProgressOverview{}, fmt.Errorf("progress overview: %w", err)
	}
	goalDates, err := s.repo.ListGoalMetDates(ctx, userID)
	if err != nil {
		return ProgressOverview{}, fmt.Errorf("progress overview: %w", err)
	}
	streak := computeStreaks(goalDates, clock.todayDate())

	byTrack := totalsByTrack(x, stats, usualTimesOf(stats))
	out := ProgressOverview{
		Tracks: make([]TrackOverview, 0, len(x.pools)),
		Today:  today,
		Streak: StreakInfo(streak),
	}
	var all answerTotals
	for _, pool := range x.pools {
		totals := byTrack[pool.Track]
		if totals == nil {
			totals = &answerTotals{}
		}
		all.merge(totals)
		out.Tracks = append(out.Tracks, trackOverview(x, pool, totals))
	}
	if clock.Settings.ShowRetention {
		out.Retention = &RetentionInfo{Actual: all.retention(), Target: clock.Settings.DesiredRetention}
	}
	return out, nil
}

// loadIndexAndStats loads every track, the user's cards and their answer
// stats since the given time (zero for all time).
func loadIndexAndStats(
	ctx context.Context, repo CardStatsRepo, loader PoolLoader, userID uuid.UUID, since time.Time,
) (*progressIndex, []learn.CardAnswerStats, error) {
	pools, err := loader.Pools(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("load pools: %w", err)
	}
	cards, err := repo.ListUserCards(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("load cards: %w", err)
	}
	stats, err := repo.ListCardAnswerStats(ctx, userID, learn.AnswerStatsFilter{Since: since})
	if err != nil {
		return nil, nil, fmt.Errorf("load answer stats: %w", err)
	}
	return newProgressIndex(pools, cards), stats, nil
}

func trackOverview(x *progressIndex, pool *learn.Pool, totals *answerTotals) TrackOverview {
	levels := x.levelCounts(pool)
	return TrackOverview{
		Track:         string(pool.Track),
		TotalItems:    len(pool.Items),
		ItemsPerLevel: levels,
		MasteredPct:   percentOf(levels.Mastered, len(pool.Items)),
		Retention30d:  totals.retention(),
		AvgAnswerMs:   totals.avgMs(),
	}
}

func (s *OverviewService) today(ctx context.Context, clock *studyClock) (TodayStats, error) {
	userID := clock.Settings.UserID
	activity, err := s.repo.GetDailyActivity(ctx, userID, clock.Today.LocalDate)
	if err != nil {
		return TodayStats{}, fmt.Errorf("today: %w", err)
	}
	dueNow, err := s.repo.CountDueCards(ctx, userID, clock.Now, nil)
	if err != nil {
		return TodayStats{}, fmt.Errorf("today: %w", err)
	}
	newKana, err := s.repo.CountItemsIntroducedSince(ctx, userID, common.SRSItemKana, clock.Today.Start)
	if err != nil {
		return TodayStats{}, fmt.Errorf("today: %w", err)
	}
	newKanji, err := s.repo.CountItemsIntroducedSince(ctx, userID, common.SRSItemKanji, clock.Today.Start)
	if err != nil {
		return TodayStats{}, fmt.Errorf("today: %w", err)
	}
	reviews, err := s.repo.CountReviewAnswersSince(ctx, userID, clock.Today.Start)
	if err != nil {
		return TodayStats{}, fmt.Errorf("today: %w", err)
	}
	return TodayStats{
		Answers:      activity.ReviewsDone,
		Correct:      activity.CorrectCount,
		ActiveMs:     activity.ActiveMs,
		NewLearned:   activity.NewLearned,
		NewKanaLeft:  max(clock.Settings.NewKanaPerDay-newKana, 0),
		NewKanjiLeft: max(clock.Settings.NewKanjiPerDay-newKanji, 0),
		ReviewsLeft:  max(clock.Settings.MaxReviewsPerDay-reviews, 0),
		GoalAnswers:  clock.Settings.DailyGoalAnswers,
		GoalMet:      activity.GoalMet,
		DueNow:       dueNow,
	}, nil
}
