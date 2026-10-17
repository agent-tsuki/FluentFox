package learn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

// fakeTestHistory filters by track as the repository does.
type fakeTestHistory []TestResult

func (f fakeTestHistory) ListTestResults(_ context.Context, _ uuid.UUID, track *common.LearningTrack) ([]TestResult, error) {
	var out []TestResult
	for _, r := range f {
		if track == nil || r.Track == *track {
			out = append(out, r)
		}
	}
	return out, nil
}

func TestTestHistoryService(t *testing.T) {
	at := func(h int) time.Time { return testNow.Add(time.Duration(h) * time.Hour) }
	result := func(track common.LearningTrack, kind common.TestKind, scope string, score, total int, passed bool, h int) TestResult {
		return TestResult{ID: uuid.New(), Track: track, Kind: kind, Scope: scope, Score: score, Total: total, Passed: passed, TakenAt: at(h)}
	}
	rows := fakeTestHistory{
		result(common.LearningTrackHiragana, common.TestKindSkip, "basic", 30, 46, false, 0),
		result(common.LearningTrackHiragana, common.TestKindSkip, "basic", 44, 46, true, 1),
		result(common.LearningTrackHiragana, common.TestKindSkip, "basic", 44, 46, true, 2),
		result(common.LearningTrackKanjiN5, common.TestKindFinal, "all", 120, 158, false, 3),
		result(common.LearningTrackHiragana, common.TestKindUnitQuiz, "ka", 5, 5, true, 4),
		result(common.LearningTrackHiragana, common.TestKindSkip, "basic", 40, 46, true, 5),
	}
	tests := []struct {
		name    string
		track   string
		want    []TestHistoryEntry
		wantErr error
	}{
		{"best score per test in the order first taken", "", []TestHistoryEntry{
			{Kind: "skip", Track: "hiragana", Scope: "basic", Score: 44, Total: 46, PassMark: 0.9, Passed: true, Attempts: 4, TakenAt: at(1), LastTakenAt: at(5)},
			{Kind: "final", Track: "kanji_n5", Scope: "all", Score: 120, Total: 158, PassMark: 0.8, Passed: false, Attempts: 1, TakenAt: at(3), LastTakenAt: at(3)},
			{Kind: "unit_quiz", Track: "hiragana", Scope: "ka", Score: 5, Total: 5, PassMark: 0.8, Passed: true, Attempts: 1, TakenAt: at(4), LastTakenAt: at(4)},
		}, nil},
		{"track filter", "kanji_n5", []TestHistoryEntry{
			{Kind: "final", Track: "kanji_n5", Scope: "all", Score: 120, Total: 158, PassMark: 0.8, Passed: false, Attempts: 1, TakenAt: at(3), LastTakenAt: at(3)},
		}, nil},
		{"no tests is an empty list", "katakana", []TestHistoryEntry{}, nil},
		{"bad track", "n4", nil, exceptions.ErrLearningTrackInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTestHistoryService(rows, zap.NewNop()).List(context.Background(), testUserID, tt.track)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(got.Tests) != len(tt.want) {
				t.Fatalf("tests = %+v, want %+v", got.Tests, tt.want)
			}
			for i := range tt.want {
				if got.Tests[i] != tt.want[i] {
					t.Errorf("test %d = %+v, want %+v", i, got.Tests[i], tt.want[i])
				}
			}
		})
	}
}
