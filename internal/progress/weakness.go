package progress

import (
	"fmt"
	"math"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
)

// weakSignals are the four inputs of the weakness score, each 0–1 except
// TimeRatio (1 is the user's usual time).
type weakSignals struct {
	Recall         float64
	LapseRate      float64
	TimeRatio      float64
	ConfusionShare float64
}

// weaknessScore follows docs/learning-plan.md section 7.4. Being faster than
// usual does not make a card look stronger, and twice the usual time or
// more counts in full.
func weaknessScore(s weakSignals) float64 {
	slowness := 0.0
	if s.TimeRatio > 0 {
		slowness = math.Min(math.Max(s.TimeRatio-1, 0), 1)
	}
	return common.WEAK_RECALL_WEIGHT*(1-s.Recall) +
		common.WEAK_LAPSE_WEIGHT*s.LapseRate +
		common.WEAK_TIME_WEIGHT*slowness +
		common.WEAK_CONFUSION_WEIGHT*s.ConfusionShare
}

// topConfusion is the item most often picked instead of a target item.
type topConfusion struct {
	Char  string
	Times int
}

// weakFacts are what the reasons and the suggestion are worded from.
type weakFacts struct {
	Lapses    int
	Recall    float64
	TimeRatio float64
	AvgMs     *int
	UsualMs   int
	Confusion *topConfusion
}

func weakReasons(f *weakFacts) []string {
	var out []string
	switch {
	case f.Lapses == 1:
		out = append(out, "Forgotten once")
	case f.Lapses > 1:
		out = append(out, fmt.Sprintf("Forgotten %d times", f.Lapses))
	}
	if isSlow(f.TimeRatio) && f.AvgMs != nil {
		out = append(out, fmt.Sprintf("Slow: %s (your usual %s)", seconds(*f.AvgMs), seconds(f.UsualMs)))
	}
	if f.Confusion != nil {
		out = append(out, fmt.Sprintf("Mixed up with %s %d×", f.Confusion.Char, f.Confusion.Times))
	}
	if f.Recall < common.WEAK_LOW_RECALL || len(out) == 0 {
		out = append(out, fmt.Sprintf("Recall chance %.0f%%", f.Recall*common.PERCENT))
	}
	return out
}

// weakSuggestion drills the mix-up when there is one, speed when the card
// is slow, and otherwise the card's unit.
func weakSuggestion(ref *cardRef, f *weakFacts) PracticeSuggestion {
	track := string(ref.Pool.Track)
	switch {
	case f.Confusion != nil:
		pair := ref.Item.Char + common.PRACTICE_PAIR_SEPARATOR + f.Confusion.Char
		return PracticeSuggestion{Track: track, Mode: string(common.PracticeModeLookalike), Pair: pair}
	case isSlow(f.TimeRatio):
		return PracticeSuggestion{Track: track, Mode: string(common.PracticeModeSpeed)}
	}
	if unit, ok := ref.Pool.Unit(ref.Item.UnitID); ok {
		return PracticeSuggestion{Track: track, Mode: string(common.PracticeModeUnit), Unit: unit.Slug}
	}
	return PracticeSuggestion{Track: track, Mode: string(common.PracticeModeWeak)}
}

func isSlow(timeRatio float64) bool {
	return timeRatio >= common.WEAK_SLOW_RATIO
}

func seconds(ms int) string {
	return fmt.Sprintf("%.1fs", (time.Duration(ms) * time.Millisecond).Seconds())
}

// lapseRate matches the weak practice ranking in internal/learn.
func lapseRate(c *learn.SRSCard) float64 {
	return float64(c.Lapses) / float64(max(c.Reps, 1))
}
