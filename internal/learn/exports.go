package learn

import (
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

// The progress package reads the same tables with the same rules: these
// expose the study day, track parsing and scheduler without copying them.

type StudyDay = dayWindow

const LocalDateLayout = localDateLayout

func StudyDayAt(now time.Time, timeZone string, startHour int) StudyDay {
	return dayWindowAt(now, timeZone, startHour)
}

func ParseTrack(raw string) (common.LearningTrack, error) {
	return parseTrack(raw)
}

func ParseOptionalTrack(raw string) (*common.LearningTrack, error) {
	return parseOptionalTrack(raw)
}

func NewSchedulerFor(s *LearningSettings) (*srs.Scheduler, error) {
	return newScheduler(s)
}

func (c *SRSCard) SRS() srs.Card {
	return toSRSCard(c)
}
