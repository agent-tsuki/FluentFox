// Package admin — service.go.
// Business logic for admin operations.
package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles admin business logic.
type Service struct {
	repo *Repository
}

// NewService constructs an admin Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetStats returns platform statistics for the admin dashboard.
func (s *Service) GetStats(ctx context.Context) (*StatsResponse, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("admin service: get stats: %w", err)
	}
	return stats, nil
}

// ListUsers returns a paginated list of all users.
func (s *Service) ListUsers(ctx context.Context, page, perPage int) ([]*AdminUserResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage
	return s.repo.ListUsers(ctx, perPage, offset)
}

// BanUser bans a user by ID and logs the action.
func (s *Service) BanUser(ctx context.Context, adminID, userID uuid.UUID, reason string) error {
	if err := s.repo.BanUser(ctx, userID); err != nil {
		return fmt.Errorf("admin service: ban user: %w", err)
	}
	return s.repo.LogAudit(ctx, adminID, "ban_user", "users", &userID,
		fmt.Sprintf(`{"reason":%q}`, reason))
}
