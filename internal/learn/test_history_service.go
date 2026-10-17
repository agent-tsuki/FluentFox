package learn

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
)

type TestHistoryRepo interface {
	ListTestResults(ctx context.Context, userID uuid.UUID, track *common.LearningTrack) ([]TestResult, error)
}

var _ TestHistoryRepo = (*Repository)(nil)

type TestHistoryService struct {
	repo   TestHistoryRepo
	logger *zap.Logger
}

func NewTestHistoryService(repo TestHistoryRepo, log *zap.Logger) *TestHistoryService {
	return &TestHistoryService{repo: repo, logger: log}
}

type testHistoryKey struct {
	Track common.LearningTrack
	Kind  common.TestKind
	Scope string
}

// List returns one entry per track, kind and scope, in the order first
// taken: the best score, and whether any attempt passed.
func (s *TestHistoryService) List(ctx context.Context, userID uuid.UUID, rawTrack string) (TestHistory, error) {
	track, err := parseOptionalTrack(rawTrack)
	if err != nil {
		return TestHistory{}, err
	}
	rows, err := s.repo.ListTestResults(ctx, userID, track)
	if err != nil {
		return TestHistory{}, fmt.Errorf("test history: %w", err)
	}
	out := TestHistory{Tests: []TestHistoryEntry{}}
	index := map[testHistoryKey]int{}
	for i := range rows {
		r := &rows[i]
		key := testHistoryKey{Track: r.Track, Kind: r.Kind, Scope: r.Scope}
		at, seen := index[key]
		if !seen {
			index[key] = len(out.Tests)
			out.Tests = append(out.Tests, toTestHistoryEntry(r))
			continue
		}
		mergeTestResult(&out.Tests[at], r)
	}
	return out, nil
}

// Rows come oldest first, so a later attempt replaces the best only with a
// strictly higher share.
func mergeTestResult(e *TestHistoryEntry, r *TestResult) {
	e.Attempts++
	e.Passed = e.Passed || r.Passed
	e.LastTakenAt = r.TakenAt
	if r.Total > 0 && (e.Total == 0 || r.Score*e.Total > e.Score*r.Total) {
		e.Score, e.Total, e.TakenAt = r.Score, r.Total, r.TakenAt
	}
}

func toTestHistoryEntry(r *TestResult) TestHistoryEntry {
	return TestHistoryEntry{
		Kind: string(r.Kind), Track: string(r.Track), Scope: r.Scope, Score: r.Score, Total: r.Total,
		PassMark: passMark(r.Kind, r.Track), Passed: r.Passed, Attempts: 1, TakenAt: r.TakenAt, LastTakenAt: r.TakenAt,
	}
}
