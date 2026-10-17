package progress

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

// In Tokyo with the day starting at 04:00, 18:00 UTC on 10 March is 03:00
// on 11 March, which still belongs to the study day of 10 March. That day
// ends at 19:00 UTC.
var tokyoNow = time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

func dueAt(t time.Time) func(*learn.SRSCard) {
	return func(c *learn.SRSCard) { c.Due = t }
}

func TestForecastServiceBucketsByLocalStudyDay(t *testing.T) {
	repo := newFakeRepo()
	repo.tz = "Asia/Tokyo"
	repo.card(common.SRSItemKana, hiraA, common.CardFacetRecognition, srs.StateReview, 5, dueAt(time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)))
	repo.card(common.SRSItemKana, hiraI, common.CardFacetRecognition, srs.StateLearning, 0, dueAt(time.Date(2026, 3, 10, 18, 59, 0, 0, time.UTC)))
	repo.card(common.SRSItemKana, kataA, common.CardFacetRecognition, srs.StateReview, 5, dueAt(time.Date(2026, 3, 10, 19, 0, 0, 0, time.UTC)))
	repo.card(common.SRSItemKanji, kanji1, common.CardFacetMeaning, srs.StateReview, 5, dueAt(time.Date(2026, 3, 12, 2, 0, 0, 0, time.UTC)))
	repo.card(common.SRSItemKanji, kanji2, common.CardFacetReading, srs.StateReview, 5, dueAt(time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)))
	repo.card(common.SRSItemKana, hiraGa, common.CardFacetRecognition, srs.StateNew, 0, dueAt(time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)))

	svc := NewForecastService(repo, zap.NewNop())
	svc.now = func() time.Time { return tokyoNow }
	got, err := svc.Get(context.Background(), testUserID, 3)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := []ForecastDay{
		{Date: "2026-03-10", ByTrack: map[string]int{"hiragana": 2}, Total: 2},
		{Date: "2026-03-11", ByTrack: map[string]int{"katakana": 1}, Total: 1},
		{Date: "2026-03-12", ByTrack: map[string]int{"kanji_n5": 1}, Total: 1},
	}
	if len(got.Days) != len(want) {
		t.Fatalf("days = %d, want %d", len(got.Days), len(want))
	}
	for i := range want {
		g, w := got.Days[i], want[i]
		if g.Date != w.Date || g.Total != w.Total || !maps.Equal(g.ByTrack, w.ByTrack) {
			t.Errorf("day %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestForecastWindows(t *testing.T) {
	tests := []struct {
		name      string
		tz        string
		now       time.Time
		days      int
		wantDates []string
		wantHours []float64
	}{
		{"Tokyo before the day start belongs to the day before", "Asia/Tokyo", tokyoNow, 2,
			[]string{"2026-03-10", "2026-03-11"}, []float64{24, 24}},
		{"daylight saving start makes a 23-hour day, not a skipped one", "America/New_York",
			time.Date(2026, 3, 7, 17, 0, 0, 0, time.UTC), 3,
			[]string{"2026-03-07", "2026-03-08", "2026-03-09"}, []float64{23, 24, 24}},
		{"unknown time zone falls back to UTC", "Not/AZone", testNow, 2,
			[]string{"2026-03-10", "2026-03-11"}, []float64{24, 24}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.tz = tt.tz
			clock, err := loadClock(context.Background(), repo, testUserID, tt.now)
			if err != nil {
				t.Fatalf("loadClock: %v", err)
			}
			windows := forecastWindows(&clock, tt.days)
			var dates []string
			var hours []float64
			for _, w := range windows {
				dates = append(dates, w.LocalDate)
				hours = append(hours, w.End.Sub(w.Start).Hours())
			}
			if !slices.Equal(dates, tt.wantDates) || !slices.Equal(hours, tt.wantHours) {
				t.Errorf("windows = %v %v, want %v %v", dates, hours, tt.wantDates, tt.wantHours)
			}
		})
	}
}

func TestDueDayIndex(t *testing.T) {
	repo := newFakeRepo()
	repo.tz = "Asia/Tokyo"
	clock, err := loadClock(context.Background(), repo, testUserID, tokyoNow)
	if err != nil {
		t.Fatalf("loadClock: %v", err)
	}
	windows := forecastWindows(&clock, 2)
	tests := []struct {
		name   string
		due    time.Time
		want   int
		wantOK bool
	}{
		{"overdue counts today", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 0, true},
		{"last minute of the study day is today", time.Date(2026, 3, 10, 18, 59, 0, 0, time.UTC), 0, true},
		{"day start belongs to the next day", time.Date(2026, 3, 10, 19, 0, 0, 0, time.UTC), 1, true},
		{"after the last day is left out", time.Date(2026, 3, 11, 19, 0, 0, 0, time.UTC), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := dueDayIndex(windows, tt.due)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("dueDayIndex = %d, %v; want %d, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
