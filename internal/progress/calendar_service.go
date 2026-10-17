package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/exceptions"
)

type CalendarService struct {
	repo   CalendarRepo
	now    func() time.Time
	logger *zap.Logger
}

func NewCalendarService(repo CalendarRepo, log *zap.Logger) *CalendarService {
	return &CalendarService{repo: repo, now: time.Now, logger: log}
}

// Get returns every local date from from to to with its activity, and the
// streaks over all time.
func (s *CalendarService) Get(ctx context.Context, userID uuid.UUID, in *CalendarInput) (ProgressCalendar, error) {
	clock, err := loadClock(ctx, s.repo, userID, s.now())
	if err != nil {
		return ProgressCalendar{}, fmt.Errorf("progress calendar: %w", err)
	}
	from, to, err := calendarRange(in, clock.todayDate())
	if err != nil {
		return ProgressCalendar{}, err
	}
	rows, err := s.repo.ListDailyActivity(ctx, userID, formatLocalDate(from), formatLocalDate(to))
	if err != nil {
		return ProgressCalendar{}, fmt.Errorf("progress calendar: %w", err)
	}
	goalDates, err := s.repo.ListGoalMetDates(ctx, userID)
	if err != nil {
		return ProgressCalendar{}, fmt.Errorf("progress calendar: %w", err)
	}
	streak := computeStreaks(goalDates, clock.todayDate())
	days := calendarDays(rows, from, to)
	active := 0
	for i := range days {
		if days[i].Answers > 0 {
			active++
		}
	}
	return ProgressCalendar{Days: days, CurrentStreak: streak.Current, LongestStreak: streak.Longest, ActiveDays: active}, nil
}

// calendarRange defaults to the year ending today.
func calendarRange(in *CalendarInput, today time.Time) (time.Time, time.Time, error) {
	to := today
	if in.To != "" {
		d, err := parseLocalDate(in.To)
		if err != nil {
			return time.Time{}, time.Time{}, exceptions.ErrProgressDateInvalid
		}
		to = d
	}
	from := to.AddDate(0, 0, 1-common.CALENDAR_DEFAULT_DAYS)
	if in.From != "" {
		d, err := parseLocalDate(in.From)
		if err != nil {
			return time.Time{}, time.Time{}, exceptions.ErrProgressDateInvalid
		}
		from = d
	}
	if from.After(to) || !from.AddDate(0, 0, common.CALENDAR_MAX_DAYS).After(to) {
		return time.Time{}, time.Time{}, exceptions.ErrProgressDateInvalid
	}
	return from, to, nil
}

// calendarDays fills every date of the range; dates without a row are empty.
func calendarDays(rows []learn.DailyActivity, from, to time.Time) []CalendarDay {
	byDate := make(map[string]*learn.DailyActivity, len(rows))
	for i := range rows {
		byDate[formatLocalDate(rows[i].LocalDate)] = &rows[i]
	}
	var out []CalendarDay
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		date := formatLocalDate(d)
		day := CalendarDay{Date: date}
		if row, ok := byDate[date]; ok {
			day.Answers = row.ReviewsDone
			day.Minutes = minutesOf(row.ActiveMs)
			day.Accuracy = ratio(row.CorrectCount, row.ReviewsDone)
			day.GoalMet = row.GoalMet
		}
		out = append(out, day)
	}
	return out
}

func minutesOf(activeMs int64) int {
	return int((time.Duration(activeMs) * time.Millisecond).Round(time.Minute) / time.Minute)
}
