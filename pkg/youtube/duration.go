package youtube

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var ErrInvalidDuration = errors.New("youtube: invalid ISO-8601 duration")

// YouTube durations use days, hours, minutes and seconds only, such as
// "PT1H2M3S" or "P1DT2H"; live streams report "P0D".
var isoDurationPattern = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

const hoursPerDay = 24

func ParseDuration(raw string) (time.Duration, error) {
	m := isoDurationPattern.FindStringSubmatch(raw)
	// "P" and "PT" alone match the pattern but name no amount.
	if m == nil || raw == "P" || raw[len(raw)-1] == 'T' {
		return 0, fmt.Errorf("%w: %q", ErrInvalidDuration, raw)
	}
	units := []time.Duration{hoursPerDay * time.Hour, time.Hour, time.Minute, time.Second}
	var total time.Duration
	for i, unit := range units {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.ParseInt(m[i+1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %q", ErrInvalidDuration, raw)
		}
		total += time.Duration(n) * unit
	}
	return total, nil
}
