package learn

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

func TestDayWindowAt(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	tests := []struct {
		name      string
		now       time.Time
		tz        string
		startHour int
		wantStart time.Time
		wantDate  string
	}{
		{
			name: "after the start hour the day began today", now: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			tz: "UTC", startHour: 4, wantStart: time.Date(2026, 3, 10, 4, 0, 0, 0, time.UTC), wantDate: "2026-03-10",
		},
		{
			name: "before the start hour the night still belongs to yesterday", now: time.Date(2026, 3, 10, 2, 0, 0, 0, time.UTC),
			tz: "UTC", startHour: 4, wantStart: time.Date(2026, 3, 9, 4, 0, 0, 0, time.UTC), wantDate: "2026-03-09",
		},
		{
			name: "the user's time zone sets the local date", now: time.Date(2026, 3, 10, 20, 0, 0, 0, time.UTC),
			tz: "Asia/Tokyo", startHour: 4, wantStart: time.Date(2026, 3, 11, 4, 0, 0, 0, tokyo), wantDate: "2026-03-11",
		},
		{
			name: "an unknown time zone falls back to UTC", now: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			tz: "Mars/Olympus", startHour: 0, wantStart: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), wantDate: "2026-03-10",
		},
		{
			name: "an empty time zone is UTC", now: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			tz: "", startHour: 4, wantStart: time.Date(2026, 3, 10, 4, 0, 0, 0, time.UTC), wantDate: "2026-03-10",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dayWindowAt(tc.now, tc.tz, tc.startHour)
			if !got.Start.Equal(tc.wantStart) {
				t.Errorf("start = %v, want %v", got.Start, tc.wantStart)
			}
			if !got.End.Equal(tc.wantStart.AddDate(0, 0, 1)) {
				t.Errorf("end = %v, want a day after start", got.End)
			}
			if got.LocalDate != tc.wantDate {
				t.Errorf("local date = %s, want %s", got.LocalDate, tc.wantDate)
			}
		})
	}
}

func TestCardIDIsStablePerUserItemAndFacet(t *testing.T) {
	base := cardID(testUserID, common.SRSItemKanji, kanjiOne, common.CardFacetMeaning)
	tests := []struct {
		name     string
		id       uuid.UUID
		wantSame bool
	}{
		{"same inputs give the same id", cardID(testUserID, common.SRSItemKanji, kanjiOne, common.CardFacetMeaning), true},
		{"another facet gives another id", cardID(testUserID, common.SRSItemKanji, kanjiOne, common.CardFacetReading), false},
		{"another user gives another id", cardID(otherUser, common.SRSItemKanji, kanjiOne, common.CardFacetMeaning), false},
		{"another item gives another id", cardID(testUserID, common.SRSItemKanji, kanjiTwo, common.CardFacetMeaning), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if (tc.id == base) != tc.wantSame {
				t.Errorf("id %s vs %s: same = %v, want %v", tc.id, base, tc.id == base, tc.wantSame)
			}
		})
	}
}

func TestPendingItems(t *testing.T) {
	kanji := loadTestPool(common.LearningTrackKanjiN5)
	hiragana := loadTestPool(common.LearningTrackHiragana)
	tests := []struct {
		name string
		pool *Pool
		keys []CardKey
		want []string
	}{
		{"no cards leaves every item pending in path order", hiragana, nil, []string{"あ", "い", "お", "じ", "ぢ", "ず"}},
		{
			"an item with its card is no longer pending", hiragana,
			[]CardKey{{ItemID: kanaA, Facet: common.CardFacetRecognition}}, []string{"い", "お", "じ", "ぢ", "ず"},
		},
		{
			"a kanji with only its meaning card is still pending", kanji,
			[]CardKey{
				{ItemID: kanjiOne, Facet: common.CardFacetMeaning},
				{ItemID: kanjiTwo, Facet: common.CardFacetMeaning}, {ItemID: kanjiTwo, Facet: common.CardFacetReading},
			},
			[]string{"一", "中", "半", "日"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, item := range pendingItems(tc.pool, tc.keys) {
				got = append(got, item.Char)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("pending = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("pending = %v, want %v", got, tc.want)
				}
			}
		})
	}
}
