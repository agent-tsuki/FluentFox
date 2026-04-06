// Package progress — service.go.
// Business logic for chapter completion, progress tracking.
package progress

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles progress business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a progress Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetOverallProgress returns the user's overall content completion stats.
func (s *Service) GetOverallProgress(ctx context.Context, userID uuid.UUID) (*OverallProgressResponse, error) {
	cc, ct, vm, vt, err := s.repo.GetOverallProgress(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("progress service: get overall: %w", err)
	}

	completionPercent := 0.0
	if ct > 0 {
		completionPercent = float64(cc) / float64(ct) * 100
	}

	return &OverallProgressResponse{
		ChaptersCompleted: cc,
		ChaptersTotal:     ct,
		VocabMastered:     vm,
		VocabTotal:        vt,
		CompletionPercent: completionPercent,
	}, nil
}

// MarkChapterStarted transitions a chapter to in_progress if not already started.
func (s *Service) MarkChapterStarted(ctx context.Context, userID, chapterID uuid.UUID) error {
	existing, _ := s.repo.GetChapterProgress(ctx, userID, chapterID)
	if existing != nil && existing.Status != "not_started" {
		return nil
	}
	return s.repo.UpsertChapterProgress(ctx, userID, chapterID, "in_progress", nil)
}

// MarkChapterCompleted records completion and score for a chapter.
func (s *Service) MarkChapterCompleted(ctx context.Context, userID, chapterID uuid.UUID, score int) error {
	if err := s.repo.UpsertChapterProgress(ctx, userID, chapterID, "completed", &score); err != nil {
		return fmt.Errorf("progress service: mark completed: %w", err)
	}
	return nil
}
