package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type UnitProgressRepo interface {
	MarkUnitStep(ctx context.Context, p *UnitProgress, step common.UnitProgressStep, at time.Time) (UnitProgress, error)
	ListUnitProgress(ctx context.Context, userID uuid.UUID) ([]UnitProgress, error)
}

var _ UnitProgressRepo = (*Repository)(nil)

type UnitProgressService struct {
	repo    UnitProgressRepo
	catalog *Catalog
	now     func() time.Time
	logger  *zap.Logger
}

func NewUnitProgressService(repo UnitProgressRepo, catalog *Catalog, log *zap.Logger) *UnitProgressService {
	return &UnitProgressService{repo: repo, catalog: catalog, now: time.Now, logger: log}
}

type trackUnit struct {
	Track common.LearningTrack
	Unit  Unit
}

// MarkStep records that the user did one step of a unit page. Doing a step
// again keeps the first time.
func (s *UnitProgressService) MarkStep(ctx context.Context, userID uuid.UUID, in *UnitProgressInput) (UnitProgressData, error) {
	step := common.UnitProgressStep(in.Body.Step)
	switch step {
	case common.UnitStepWatched, common.UnitStepTraced, common.UnitStepWritten, common.UnitStepPractised:
	default:
		return UnitProgressData{}, exceptions.ErrUnitProgressStepInvalid
	}
	tu, err := s.findUnit(ctx, common.UnitType(in.Type), in.ID, in.Track)
	if err != nil {
		return UnitProgressData{}, err
	}
	row, err := s.repo.MarkUnitStep(ctx, &UnitProgress{UserID: userID, UnitType: tu.Unit.Type, UnitID: tu.Unit.ID}, step, s.now().UTC())
	if err != nil {
		return UnitProgressData{}, fmt.Errorf("mark unit step: %w", err)
	}
	return toUnitProgressData(&tu, &row), nil
}

// List returns every unit of the track (or of all tracks) in path order,
// with the user's progress where there is any.
func (s *UnitProgressService) List(ctx context.Context, userID uuid.UUID, rawTrack string) (UnitProgressList, error) {
	track, err := parseOptionalTrack(rawTrack)
	if err != nil {
		return UnitProgressList{}, err
	}
	pools, err := s.catalog.Pools(ctx, track)
	if err != nil {
		return UnitProgressList{}, fmt.Errorf("list unit progress: %w", err)
	}
	rows, err := s.repo.ListUnitProgress(ctx, userID)
	if err != nil {
		return UnitProgressList{}, fmt.Errorf("list unit progress: %w", err)
	}
	byUnit := make(map[uuid.UUID]*UnitProgress, len(rows))
	for i := range rows {
		byUnit[rows[i].UnitID] = &rows[i]
	}
	out := UnitProgressList{Units: []UnitProgressData{}}
	for _, pool := range pools {
		for _, u := range pool.Units {
			row := byUnit[u.ID]
			if row == nil {
				row = &UnitProgress{}
			}
			out.Units = append(out.Units, toUnitProgressData(&trackUnit{Track: pool.Track, Unit: u}, row))
		}
	}
	return out, nil
}

// findUnit resolves a unit slug (or id) among the tracks of the unit type.
// Hiragana and katakana share slugs such as "ka", so a slug found in both
// needs the track.
func (s *UnitProgressService) findUnit(ctx context.Context, unitType common.UnitType, id, rawTrack string) (trackUnit, error) {
	tracks, err := unitTracks(unitType, rawTrack)
	if err != nil {
		return trackUnit{}, err
	}
	var found []trackUnit
	for _, track := range tracks {
		pool, err := s.catalog.Load(ctx, track)
		if err != nil {
			return trackUnit{}, fmt.Errorf("find unit: %w", err)
		}
		for _, u := range pool.Units {
			if u.Slug == id || u.ID.String() == id {
				found = append(found, trackUnit{Track: track, Unit: u})
			}
		}
	}
	switch len(found) {
	case 0:
		return trackUnit{}, exceptions.ErrUnitNotFound
	case 1:
		return found[0], nil
	default:
		return trackUnit{}, exceptions.ErrUnitAmbiguous
	}
}

// unitTracks lists the tracks with units of the type, or only the given one.
func unitTracks(unitType common.UnitType, rawTrack string) ([]common.LearningTrack, error) {
	only, err := parseOptionalTrack(rawTrack)
	if err != nil {
		return nil, err
	}
	var out []common.LearningTrack
	for _, track := range common.LEARNING_TRACKS {
		if common.LEARNING_TRACK_UNIT_TYPE[track] == unitType && (only == nil || *only == track) {
			out = append(out, track)
		}
	}
	return out, nil
}

func toUnitProgressData(tu *trackUnit, p *UnitProgress) UnitProgressData {
	return UnitProgressData{
		UnitType:    string(tu.Unit.Type),
		UnitID:      tu.Unit.ID,
		UnitSlug:    tu.Unit.Slug,
		Track:       string(tu.Track),
		WatchedAt:   p.WatchedAt,
		TracedAt:    p.TracedAt,
		WrittenAt:   p.WrittenAt,
		PractisedAt: p.PractisedAt,
		QuizBestPct: p.QuizBestPct,
		CompletedAt: p.CompletedAt,
	}
}
