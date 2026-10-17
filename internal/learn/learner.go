package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

const localDateLayout = "2006-01-02"

// dayWindow is the user's study day: it starts at day_starts_at_hour in the
// user's time zone, so a late-night session still counts for the evening.
type dayWindow struct {
	Start     time.Time
	End       time.Time
	LocalDate string
}

// An unknown or empty time zone falls back to UTC rather than failing the
// request, because the zone is free text in users_settings.
func dayWindowAt(now time.Time, timeZone string, startHour int) dayWindow {
	loc, err := time.LoadLocation(timeZone)
	if err != nil || timeZone == "" {
		loc = time.UTC
	}
	shifted := now.In(loc).Add(-time.Duration(startHour) * time.Hour)
	y, m, d := shifted.Date()
	start := time.Date(y, m, d, startHour, 0, 0, 0, loc)
	return dayWindow{
		Start:     start,
		End:       time.Date(y, m, d+1, startHour, 0, 0, 0, loc),
		LocalDate: start.Format(localDateLayout),
	}
}

type learner struct {
	UserID   uuid.UUID
	Settings LearningSettings
	Day      dayWindow
}

func loadLearner(ctx context.Context, repo LearnerRepo, userID uuid.UUID, now time.Time) (learner, error) {
	settings, err := repo.EnsureSettings(ctx, userID)
	if err != nil {
		return learner{}, fmt.Errorf("load learner: %w", err)
	}
	tz, err := repo.GetTimeZone(ctx, userID)
	if err != nil {
		return learner{}, fmt.Errorf("load learner: %w", err)
	}
	return learner{UserID: userID, Settings: settings, Day: dayWindowAt(now, tz, settings.DayStartsAtHour)}, nil
}

func (l *learner) newPerDay(itemType common.SRSItemType) int {
	if itemType == common.SRSItemKanji {
		return l.Settings.NewKanjiPerDay
	}
	return l.Settings.NewKanaPerDay
}

func newScheduler(s *LearningSettings) (*srs.Scheduler, error) {
	sched, err := srs.NewScheduler(srs.Params{DesiredRetention: s.DesiredRetention, Weights: s.FSRSWeights})
	if err != nil {
		return nil, fmt.Errorf("build scheduler for %s: %w", s.UserID, err)
	}
	return sched, nil
}

func parseTrack(raw string) (common.LearningTrack, error) {
	track := common.LearningTrack(raw)
	if _, ok := common.LEARNING_TRACK_ITEM_TYPE[track]; !ok {
		return "", exceptions.ErrLearningTrackInvalid
	}
	return track, nil
}

// An empty value means every track.
func parseOptionalTrack(raw string) (*common.LearningTrack, error) {
	if raw == "" {
		return nil, nil
	}
	track, err := parseTrack(raw)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func cardID(userID uuid.UUID, itemType common.SRSItemType, itemID uuid.UUID, facet common.CardFacet) uuid.UUID {
	name := userID.String() + "/" + string(itemType) + "/" + itemID.String() + "/" + string(facet)
	return uuid.NewSHA1(common.SRS_CARD_ID_NAMESPACE, []byte(name))
}

// pendingItems lists the items of a pool, in path order, that still miss a
// card for at least one facet.
func pendingItems(pool *Pool, keys []CardKey) []*Item {
	have := make(map[uuid.UUID]map[common.CardFacet]bool, len(keys))
	for _, k := range keys {
		if have[k.ItemID] == nil {
			have[k.ItemID] = map[common.CardFacet]bool{}
		}
		have[k.ItemID][k.Facet] = true
	}
	var out []*Item
	for i := range pool.Items {
		item := &pool.Items[i]
		if len(missingFacets(item, have[item.ID])) > 0 {
			out = append(out, item)
		}
	}
	return out
}

func missingFacets(item *Item, have map[common.CardFacet]bool) []common.CardFacet {
	var out []common.CardFacet
	for _, f := range common.SRS_ITEM_FACETS[item.Type] {
		if !have[f] {
			out = append(out, f)
		}
	}
	return out
}
