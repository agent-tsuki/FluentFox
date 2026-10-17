package learn

import (
	"math"
	"slices"
	"time"
)

// TypingSample is one correct typed answer. Time after the first key press is
// treated as typing, which gives the user's typing speed per character.
type TypingSample struct {
	Active     time.Duration
	FirstInput time.Duration
	Chars      int
}

// BaselineFrom expects the user's last BASELINE_WINDOW correct answers of one
// type. For typed types pass ThinkingTime values so they match what Grade
// scores. Non-positive durations are skipped because their log is undefined.
func BaselineFrom(durations []time.Duration) Baseline {
	logs := make([]float64, 0, len(durations))
	for _, d := range durations {
		if d > 0 {
			logs = append(logs, math.Log(durationToMs(d)))
		}
	}
	if len(logs) == 0 {
		return Baseline{}
	}
	median := medianOf(logs)
	deviations := make([]float64, len(logs))
	for i, l := range logs {
		deviations[i] = math.Abs(l - median)
	}
	return Baseline{MedianLogMs: median, MADLogMs: medianOf(deviations), Count: len(logs)}
}

func TypingPerChar(samples []TypingSample) time.Duration {
	perChar := make([]float64, 0, len(samples))
	for _, s := range samples {
		typing := s.Active - s.FirstInput
		if s.Chars > 0 && typing > 0 {
			perChar = append(perChar, float64(typing)/float64(s.Chars))
		}
	}
	if len(perChar) == 0 {
		return 0
	}
	return time.Duration(math.Round(medianOf(perChar)))
}

func medianOf(values []float64) float64 {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}
