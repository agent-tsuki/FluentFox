package learn

import (
	"context"
	"testing"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
)

type countingContent struct {
	*fakeContent
	kanaUnitCalls int
	empty         bool
}

func (c *countingContent) ListKanaUnits(ctx context.Context, script common.KanaScript) ([]content.KanaUnit, error) {
	c.kanaUnitCalls++
	return c.fakeContent.ListKanaUnits(ctx, script)
}

func (c *countingContent) ListKana(ctx context.Context, script common.KanaScript) ([]content.Kana, error) {
	if c.empty {
		return nil, nil
	}
	return c.fakeContent.ListKana(ctx, script)
}

func TestCatalogCachesLoadedTracks(t *testing.T) {
	start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		empty     bool
		secondAt  time.Time
		wantCalls int
	}{
		{"a second load within the TTL reuses the pool", false, start.Add(time.Minute), 1},
		{"a load after the TTL reads the content again", false, start.Add(common.CONTENT_CACHE_TTL), 2},
		{"an empty pool is not cached", true, start.Add(time.Minute), 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := &countingContent{fakeContent: &fakeContent{}, empty: tc.empty}
			catalog := NewCatalog(reader)
			now := start
			catalog.now = func() time.Time { return now }

			first, err := catalog.Load(context.Background(), common.LearningTrackHiragana)
			if err != nil {
				t.Fatalf("first load: %v", err)
			}
			now = tc.secondAt
			second, err := catalog.Load(context.Background(), common.LearningTrackHiragana)
			if err != nil {
				t.Fatalf("second load: %v", err)
			}
			if reader.kanaUnitCalls != tc.wantCalls {
				t.Errorf("content reads = %d, want %d", reader.kanaUnitCalls, tc.wantCalls)
			}
			if tc.wantCalls == 1 && first != second {
				t.Error("cached load returned a different pool")
			}
		})
	}
}
