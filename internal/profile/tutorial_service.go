package profile

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

// TutorialService reads and records the learner's in-app tutorials (the
// first-visit welcome tour). Unknown keys are refused, so a typo in the
// frontend can't create rows nobody reads.
type TutorialService struct {
	repo   TutorialRepo
	logger *zap.Logger
	now    func() time.Time
}

func NewTutorialService(repo TutorialRepo, log *zap.Logger) *TutorialService {
	return &TutorialService{repo: repo, logger: log, now: time.Now}
}

// List returns every known tutorial; one with no row is pending at version 1.
func (s *TutorialService) List(ctx context.Context, userID uuid.UUID) (TutorialsResponse, error) {
	rows, err := s.repo.ListUserTutorials(ctx, userID)
	if err != nil {
		return TutorialsResponse{}, err
	}
	byKey := make(map[string]users.UserTutorial, len(rows))
	for _, r := range rows {
		byKey[r.TutorialKey] = r
	}
	out := make([]TutorialResponse, 0, len(common.TUTORIAL_KEYS))
	for _, key := range common.TUTORIAL_KEYS {
		row, ok := byKey[key]
		if !ok {
			out = append(out, TutorialResponse{Key: key, Status: string(common.TutorialPending), Version: 1})
			continue
		}
		out = append(out, toTutorialResponse(&row))
	}
	return TutorialsResponse{Tutorials: out}, nil
}

// Update records the learner's latest state for one tutorial.
func (s *TutorialService) Update(ctx context.Context, userID uuid.UUID, key string, req *UpdateTutorialRequest) (TutorialResponse, error) {
	if !slices.Contains(common.TUTORIAL_KEYS, key) {
		return TutorialResponse{}, exceptions.ErrTutorialNotFound
	}
	now := s.now().UTC()
	row := users.UserTutorial{
		UserID:      userID,
		TutorialKey: key,
		Status:      common.TutorialStatus(req.Status),
		Version:     req.Version,
		LastStep:    req.LastStep,
		UpdatedAt:   now,
	}
	if row.Status == common.TutorialCompleted {
		row.CompletedAt = &now
	}
	saved, err := s.repo.UpsertUserTutorial(ctx, &row)
	if err != nil {
		return TutorialResponse{}, err
	}
	return toTutorialResponse(&saved), nil
}

func toTutorialResponse(r *users.UserTutorial) TutorialResponse {
	updated := r.UpdatedAt
	return TutorialResponse{
		Key:         r.TutorialKey,
		Status:      string(r.Status),
		Version:     r.Version,
		LastStep:    r.LastStep,
		CompletedAt: r.CompletedAt,
		UpdatedAt:   &updated,
	}
}
