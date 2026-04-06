// Package streak — service.go.
// Business logic: recording activity, incrementing/resetting streaks,
// detecting streak death vs. freeze use.
package streak

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles streak business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a streak Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetStreak returns the current streak state for a user.
func (s *Service) GetStreak(ctx context.Context, userID uuid.UUID) (*StreakResponse, error) {
	streak, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("streak service: get streak: %w", err)
	}

	isAlive := isStreakAlive(streak)
	var lastDate *string
	if streak.LastActivityDate != nil {
		d := streak.LastActivityDate.Format("2006-01-02")
		lastDate = &d
	}

	return &StreakResponse{
		CurrentStreak:    streak.CurrentStreak,
		LongestStreak:    streak.LongestStreak,
		LastActivityDate: lastDate,
		FreezeCount:      streak.FreezeCount,
		IsAlive:          isAlive,
	}, nil
}

// RecordActivity updates the streak for today's activity.
// Increments streak if activity is on consecutive days.
// Resets streak if more than one day was missed and no freezes remain.
func (s *Service) RecordActivity(ctx context.Context, userID uuid.UUID, activityType string) error {
	streak, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		// First activity ever — create the streak record.
		streak = &UserStreak{UserID: userID}
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	if streak.LastActivityDate != nil {
		lastDay := streak.LastActivityDate.Truncate(24 * time.Hour)
		daysSince := int(today.Sub(lastDay).Hours() / 24)

		switch {
		case daysSince == 0:
			// Already recorded today — no change.
			return nil
		case daysSince == 1:
			// Consecutive day — increment.
			streak.CurrentStreak++
		default:
			// Missed days — check for freeze.
			missed := daysSince - 1
			if streak.FreezeCount >= missed {
				streak.FreezeCount -= missed
				streak.CurrentStreak++
			} else {
				// Streak broken.
				streak.CurrentStreak = 1
				streak.FreezeCount = 0
			}
		}
	} else {
		streak.CurrentStreak = 1
	}

	streak.LastActivityDate = &today
	if streak.CurrentStreak > streak.LongestStreak {
		streak.LongestStreak = streak.CurrentStreak
	}

	if err := s.repo.UpsertStreak(ctx, streak); err != nil {
		return fmt.Errorf("streak service: record activity: %w", err)
	}

	return s.repo.LogActivity(ctx, userID, activityType)
}

// isStreakAlive returns true if the streak is still valid (activity within 2 days).
func isStreakAlive(s *UserStreak) bool {
	if s.LastActivityDate == nil {
		return false
	}
	now := time.Now().Truncate(24 * time.Hour)
	last := s.LastActivityDate.Truncate(24 * time.Hour)
	daysSince := int(now.Sub(last).Hours() / 24)
	return daysSince <= 1 || (daysSince-1) <= s.FreezeCount
}
