package learn

import (
	"math"
	"slices"
	"testing"
	"time"
)

func TestBaselineFrom(t *testing.T) {
	ln := math.Log
	tests := []struct {
		name       string
		durations  []time.Duration
		wantMedian float64
		wantMAD    float64
		wantCount  int
	}{
		{name: "no answers gives an empty baseline", durations: nil},
		{name: "one answer has zero spread", durations: []time.Duration{time.Second},
			wantMedian: ln(1000), wantMAD: 0, wantCount: 1},
		{name: "odd count uses the middle value", durations: []time.Duration{4 * time.Second, time.Second, 2 * time.Second},
			wantMedian: ln(2000), wantMAD: ln(2), wantCount: 3},
		{name: "even count averages the middle logs", durations: []time.Duration{time.Second, 4 * time.Second},
			wantMedian: ln(2000), wantMAD: ln(2), wantCount: 2},
		{name: "one outlier does not move the median or spread", durations: []time.Duration{time.Second, time.Second, time.Second, time.Minute},
			wantMedian: ln(1000), wantMAD: 0, wantCount: 4},
		{name: "non-positive durations are skipped", durations: []time.Duration{0, -time.Second, time.Second},
			wantMedian: ln(1000), wantMAD: 0, wantCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BaselineFrom(tt.durations)
			if got.Count != tt.wantCount {
				t.Fatalf("count = %d, want %d", got.Count, tt.wantCount)
			}
			if math.Abs(got.MedianLogMs-tt.wantMedian) > 1e-9 || math.Abs(got.MADLogMs-tt.wantMAD) > 1e-9 {
				t.Fatalf("median/MAD = %v/%v, want %v/%v", got.MedianLogMs, got.MADLogMs, tt.wantMedian, tt.wantMAD)
			}
		})
	}

	t.Run("does not reorder the caller's slice", func(t *testing.T) {
		in := []time.Duration{3 * time.Second, time.Second, 2 * time.Second}
		want := slices.Clone(in)
		BaselineFrom(in)
		if !slices.Equal(in, want) {
			t.Fatalf("input changed: %v", in)
		}
	})
}

func TestTypingPerChar(t *testing.T) {
	tests := []struct {
		name    string
		samples []TypingSample
		want    time.Duration
	}{
		{name: "no samples is zero", samples: nil, want: 0},
		{name: "time after first input is spread over the characters",
			samples: []TypingSample{{Active: 3 * time.Second, FirstInput: time.Second, Chars: 4}}, want: 500 * time.Millisecond},
		{name: "median of several answers",
			samples: []TypingSample{
				{Active: 2 * time.Second, FirstInput: time.Second, Chars: 10},
				{Active: 3 * time.Second, FirstInput: time.Second, Chars: 10},
				{Active: 9 * time.Second, FirstInput: time.Second, Chars: 2},
			}, want: 200 * time.Millisecond},
		{name: "answers without characters or typing time are skipped",
			samples: []TypingSample{
				{Active: 3 * time.Second, FirstInput: time.Second, Chars: 0},
				{Active: time.Second, FirstInput: 2 * time.Second, Chars: 3},
				{Active: 2 * time.Second, FirstInput: time.Second, Chars: 5},
			}, want: 200 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TypingPerChar(tt.samples); got != tt.want {
				t.Fatalf("TypingPerChar = %v, want %v", got, tt.want)
			}
		})
	}
}
