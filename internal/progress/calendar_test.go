package progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestCalendarRange(t *testing.T) {
	today := date("2026-03-10")
	tests := []struct {
		name     string
		in       CalendarInput
		wantFrom string
		wantTo   string
		wantErr  error
	}{
		{"defaults to the year ending today", CalendarInput{}, "2025-03-11", "2026-03-10", nil},
		{"from only runs to today", CalendarInput{From: "2026-03-01"}, "2026-03-01", "2026-03-10", nil},
		{"to only covers the year before it", CalendarInput{To: "2026-01-31"}, "2025-02-01", "2026-01-31", nil},
		{"single day", CalendarInput{From: "2026-03-05", To: "2026-03-05"}, "2026-03-05", "2026-03-05", nil},
		{"366 days is the most", CalendarInput{From: "2025-03-10", To: "2026-03-10"}, "2025-03-10", "2026-03-10", nil},
		{"367 days is too many", CalendarInput{From: "2025-03-09", To: "2026-03-10"}, "", "", exceptions.ErrProgressDateInvalid},
		{"from after to", CalendarInput{From: "2026-03-11", To: "2026-03-10"}, "", "", exceptions.ErrProgressDateInvalid},
		{"bad from", CalendarInput{From: "10/03/2026"}, "", "", exceptions.ErrProgressDateInvalid},
		{"bad to", CalendarInput{To: "2026-02-30"}, "", "", exceptions.ErrProgressDateInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := calendarRange(&tt.in, today)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if formatLocalDate(from) != tt.wantFrom || formatLocalDate(to) != tt.wantTo {
				t.Errorf("range = %s..%s, want %s..%s", formatLocalDate(from), formatLocalDate(to), tt.wantFrom, tt.wantTo)
			}
		})
	}
}

func TestCalendarServiceFillsDaysAndStreaks(t *testing.T) {
	repo := newFakeRepo()
	repo.activity = []learn.DailyActivity{
		{LocalDate: date("2026-03-07"), ReviewsDone: 30, CorrectCount: 24, ActiveMs: 600_000, GoalMet: true},
		{LocalDate: date("2026-03-09"), ReviewsDone: 20, CorrectCount: 15, ActiveMs: 89_000, GoalMet: true},
		{LocalDate: date("2026-03-10"), ReviewsDone: 4, CorrectCount: 4, ActiveMs: 20_000},
	}
	repo.goalDates = []time.Time{date("2026-03-01"), date("2026-03-02"), date("2026-03-03"), date("2026-03-07"), date("2026-03-09")}
	svc := NewCalendarService(repo, zap.NewNop())
	svc.now = func() time.Time { return testNow }

	got, err := svc.Get(context.Background(), testUserID, &CalendarInput{From: "2026-03-07", To: "2026-03-10"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	accuracy := func(v float64) *float64 { return &v }
	want := []CalendarDay{
		{Date: "2026-03-07", Answers: 30, Minutes: 10, Accuracy: accuracy(0.8), GoalMet: true},
		{Date: "2026-03-08"},
		{Date: "2026-03-09", Answers: 20, Minutes: 1, Accuracy: accuracy(0.75), GoalMet: true},
		{Date: "2026-03-10", Answers: 4, Minutes: 0, Accuracy: accuracy(1)},
	}
	if len(got.Days) != len(want) {
		t.Fatalf("days = %d, want %d", len(got.Days), len(want))
	}
	for i := range want {
		g, w := got.Days[i], want[i]
		sameAccuracy := (g.Accuracy == nil) == (w.Accuracy == nil) && (g.Accuracy == nil || *g.Accuracy == *w.Accuracy)
		if g.Date != w.Date || g.Answers != w.Answers || g.Minutes != w.Minutes || g.GoalMet != w.GoalMet || !sameAccuracy {
			t.Errorf("day %d = %+v, want %+v", i, g, w)
		}
	}
	if got.CurrentStreak != 1 || got.LongestStreak != 3 || got.ActiveDays != 3 {
		t.Errorf("streaks = current %d, longest %d, active %d; want 1, 3, 3", got.CurrentStreak, got.LongestStreak, got.ActiveDays)
	}
}
