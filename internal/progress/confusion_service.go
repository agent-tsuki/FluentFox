package progress

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
)

type ConfusionService struct {
	repo   ConfusionRepo
	pools  PoolLoader
	logger *zap.Logger
}

func NewConfusionService(repo ConfusionRepo, pools PoolLoader, log *zap.Logger) *ConfusionService {
	return &ConfusionService{repo: repo, pools: pools, logger: log}
}

// Get returns the user's most frequent mix-ups, most frequent first. A pair
// belongs to the track of the item that was asked.
func (s *ConfusionService) Get(ctx context.Context, userID uuid.UUID, in *ConfusionsInput) (ConfusionsResponse, error) {
	track, err := learn.ParseOptionalTrack(in.Track)
	if err != nil {
		return ConfusionsResponse{}, fmt.Errorf("confusions: %w", err)
	}
	pools, err := s.pools.Pools(ctx, track)
	if err != nil {
		return ConfusionsResponse{}, fmt.Errorf("confusions: %w", err)
	}
	rows, err := s.repo.ListConfusions(ctx, userID)
	if err != nil {
		return ConfusionsResponse{}, fmt.Errorf("confusions: %w", err)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = common.CONFUSIONS_DEFAULT_LIMIT
	}
	out := ConfusionsResponse{Pairs: []ConfusionPair{}}
	for i := range rows {
		if len(out.Pairs) == limit {
			break
		}
		if pair, ok := confusionPair(pools, &rows[i]); ok {
			out.Pairs = append(out.Pairs, pair)
		}
	}
	return out, nil
}

func confusionPair(pools learn.Pools, r *learn.UserConfusion) (ConfusionPair, bool) {
	pool, target, ok := pools.Find(r.ItemType, r.TargetItemID)
	if !ok {
		return ConfusionPair{}, false
	}
	chosen, ok := pool.Item(r.ChosenItemID)
	if !ok {
		return ConfusionPair{}, false
	}
	return ConfusionPair{
		ItemType: string(r.ItemType), Track: string(pool.Track),
		Target: ConfusionSide{ItemID: target.ID, Char: target.Char},
		Chosen: ConfusionSide{ItemID: chosen.ID, Char: chosen.Char},
		Times:  r.Times, LastAt: r.LastAt,
	}, true
}
