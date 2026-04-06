// Package xp — service.go.
// Business logic: awarding XP, computing levels, leaderboard.
package xp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles XP business logic.
type Service struct {
	repo *Repository
}

// NewService constructs an XP Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetXP returns the user's current XP and level.
func (s *Service) GetXP(ctx context.Context, userID uuid.UUID) (*XPResponse, error) {
	userXP, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("xp service: get xp: %w", err)
	}

	levels, err := s.repo.GetLevelConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("xp service: get level config: %w", err)
	}

	xpToNext, progress := computeLevelProgress(userXP.Total, levels)

	return &XPResponse{
		Total:         userXP.Total,
		Level:         userXP.Level,
		XPToNextLevel: xpToNext,
		Progress:      progress,
	}, nil
}

// Award adds XP to a user for the given source and recomputes their level.
func (s *Service) Award(ctx context.Context, userID uuid.UUID, amount int, source string, sourceID *uuid.UUID) error {
	if amount <= 0 {
		return nil
	}

	newTotal, err := s.repo.AddXP(ctx, userID, amount, source, sourceID)
	if err != nil {
		return fmt.Errorf("xp service: award: %w", err)
	}

	levels, err := s.repo.GetLevelConfig(ctx)
	if err != nil {
		return fmt.Errorf("xp service: get level config for recalc: %w", err)
	}

	newLevel := computeLevel(newTotal, levels)
	return s.repo.UpdateLevel(ctx, userID, newLevel)
}

// GetLeaderboard returns the top 50 users by XP.
func (s *Service) GetLeaderboard(ctx context.Context) ([]*LeaderboardEntry, error) {
	return s.repo.GetLeaderboard(ctx, 50)
}

// computeLevel returns the level for a given total XP.
func computeLevel(total int, levels []*XPLevelConfig) int {
	current := 1
	for _, l := range levels {
		if total >= l.XPRequired {
			current = l.Level
		}
	}
	return current
}

// computeLevelProgress returns XP remaining to next level and progress percent.
func computeLevelProgress(total int, levels []*XPLevelConfig) (xpToNext int, progress float64) {
	var currentThreshold, nextThreshold int
	for i, l := range levels {
		if total >= l.XPRequired {
			currentThreshold = l.XPRequired
			if i+1 < len(levels) {
				nextThreshold = levels[i+1].XPRequired
			}
		}
	}
	if nextThreshold == 0 {
		return 0, 100.0
	}
	xpToNext = nextThreshold - total
	levelRange := nextThreshold - currentThreshold
	if levelRange > 0 {
		progress = float64(total-currentThreshold) / float64(levelRange) * 100
	}
	return xpToNext, progress
}
