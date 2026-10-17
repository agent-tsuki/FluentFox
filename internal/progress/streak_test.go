package progress

import (
	"testing"
	"time"
)

func TestComputeStreaks(t *testing.T) {
	dates := func(ds ...string) []time.Time {
		out := make([]time.Time, len(ds))
		for i, d := range ds {
			out[i] = date(d)
		}
		return out
	}
	today := date("2026-03-10")
	tests := []struct {
		name  string
		goals []time.Time
		want  streaks
	}{
		{"no goal days", nil, streaks{}},
		{"today met counts", dates("2026-03-08", "2026-03-09", "2026-03-10"), streaks{Current: 3, Longest: 3}},
		{"unfinished today keeps yesterday's streak", dates("2026-03-08", "2026-03-09"), streaks{Current: 2, Longest: 2}},
		{"missed yesterday breaks the streak", dates("2026-03-07", "2026-03-08"), streaks{Current: 0, Longest: 2}},
		{"only today", dates("2026-03-10"), streaks{Current: 1, Longest: 1}},
		{"gap splits runs and longest is kept", dates("2026-02-01", "2026-02-02", "2026-02-03", "2026-02-04", "2026-03-09", "2026-03-10"),
			streaks{Current: 2, Longest: 4}},
		{"run across a month end", dates("2026-02-27", "2026-02-28", "2026-03-01"), streaks{Current: 0, Longest: 3}},
		{"dates in another location still match", []time.Time{
			time.Date(2026, 3, 9, 0, 0, 0, 0, time.FixedZone("JST", 9*3600)),
			time.Date(2026, 3, 10, 0, 0, 0, 0, time.FixedZone("JST", 9*3600)),
		}, streaks{Current: 2, Longest: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeStreaks(tt.goals, today); got != tt.want {
				t.Errorf("computeStreaks = %+v, want %+v", got, tt.want)
			}
		})
	}
}
