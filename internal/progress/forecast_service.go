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

type ForecastService struct {
	repo   ForecastRepo
	now    func() time.Time
	logger *zap.Logger
}

func NewForecastService(repo ForecastRepo, log *zap.Logger) *ForecastService {
	return &ForecastService{repo: repo, now: time.Now, logger: log}
}

// Get counts the reviews due on each of the next days, per track. Cards
// already due count on today, as do learning steps due later today.
func (s *ForecastService) Get(ctx context.Context, userID uuid.UUID, days int) (ProgressForecast, error) {
	if days <= 0 {
		days = common.FORECAST_DEFAULT_DAYS
	}
	clock, err := loadClock(ctx, s.repo, userID, s.now())
	if err != nil {
		return ProgressForecast{}, fmt.Errorf("progress forecast: %w", err)
	}
	windows := forecastWindows(&clock, days)
	out := ProgressForecast{Days: make([]ForecastDay, len(windows))}
	for i := range windows {
		out.Days[i] = ForecastDay{Date: windows[i].LocalDate, ByTrack: map[string]int{}}
	}
	end := windows[len(windows)-1].End
	for _, track := range common.LEARNING_TRACKS {
		cards, err := s.repo.ListDueCards(ctx, userID, end, &track)
		if err != nil {
			return ProgressForecast{}, fmt.Errorf("progress forecast: %w", err)
		}
		for i := range cards {
			day, ok := dueDayIndex(windows, cards[i].Due)
			if !ok {
				continue
			}
			out.Days[day].ByTrack[string(track)]++
			out.Days[day].Total++
		}
	}
	return out, nil
}

// forecastWindows lists the study days from today on. The next day is
// looked up from the middle of it rather than from its start, because the
// day window steps back the start hour in elapsed time, which lands in the
// day before when a daylight saving change makes a day 23 hours long.
func forecastWindows(clock *studyClock, days int) []learn.StudyDay {
	out := make([]learn.StudyDay, 0, days)
	day := clock.Today
	for range days {
		out = append(out, day)
		day = clock.dayOf(day.End.Add(common.STUDY_DAY_MIDPOINT))
	}
	return out
}

// dueDayIndex finds the window a due time falls in; anything before the end
// of the first window (overdue included) belongs to it.
func dueDayIndex(windows []learn.StudyDay, due time.Time) (int, bool) {
	for i := range windows {
		if due.Before(windows[i].End) {
			return i, true
		}
	}
	return 0, false
}
