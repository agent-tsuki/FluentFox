package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/learn"
)

// studyClock finds the user's study day for any instant. It keeps the time
// zone, which forecast and calendar need for days other than today.
type studyClock struct {
	Now      time.Time
	Settings learn.LearningSettings
	Today    learn.StudyDay
	timeZone string
}

func loadClock(ctx context.Context, repo learn.LearnerRepo, userID uuid.UUID, now time.Time) (studyClock, error) {
	settings, err := repo.EnsureSettings(ctx, userID)
	if err != nil {
		return studyClock{}, fmt.Errorf("load study clock: %w", err)
	}
	tz, err := repo.GetTimeZone(ctx, userID)
	if err != nil {
		return studyClock{}, fmt.Errorf("load study clock: %w", err)
	}
	c := studyClock{Now: now, Settings: settings, timeZone: tz}
	c.Today = c.dayOf(now)
	return c, nil
}

func (c *studyClock) dayOf(t time.Time) learn.StudyDay {
	return learn.StudyDayAt(t, c.timeZone, c.Settings.DayStartsAtHour)
}

// todayDate is today's local date as midnight UTC, the form
// daily_activity.local_date is read back in.
func (c *studyClock) todayDate() time.Time {
	y, m, d := c.Today.Start.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func parseLocalDate(s string) (time.Time, error) {
	d, err := time.Parse(learn.LocalDateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse local date %q: %w", s, err)
	}
	return d, nil
}

func formatLocalDate(d time.Time) string {
	return d.Format(learn.LocalDateLayout)
}
