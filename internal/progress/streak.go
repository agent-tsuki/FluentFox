package progress

import (
	"time"
)

type streaks struct {
	Current int
	Longest int
}

// computeStreaks counts consecutive local days with the goal met. Today
// counts once its goal is met, but an unfinished today does not break the
// streak that ended yesterday. Dates are compared as YYYY-MM-DD because the
// driver may hand dates back in any location.
func computeStreaks(goalDates []time.Time, today time.Time) streaks {
	met := make(map[string]bool, len(goalDates))
	var out streaks
	run := 0
	prev := ""
	for _, d := range goalDates {
		key := formatLocalDate(d)
		met[key] = true
		if run > 0 && key == nextDate(prev) {
			run++
		} else {
			run = 1
		}
		prev = key
		out.Longest = max(out.Longest, run)
	}
	day := formatLocalDate(today)
	if !met[day] {
		day = prevDate(day)
	}
	for met[day] {
		out.Current++
		day = prevDate(day)
	}
	return out
}

func nextDate(date string) string { return shiftDate(date, 1) }
func prevDate(date string) string { return shiftDate(date, -1) }

// An unparsable date shifts to "", which matches no day.
func shiftDate(date string, days int) string {
	d, err := parseLocalDate(date)
	if err != nil {
		return ""
	}
	return formatLocalDate(d.AddDate(0, 0, days))
}
