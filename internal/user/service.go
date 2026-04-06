// Package user — service.go.
// Owns user-domain business logic: fetching current user, updating profile,
// settings, and changing password.
// Must never know about HTTP. Must never write SQL.
package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service handles user-domain business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a user Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetMe returns the authenticated user's public data.
func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user service: get me: %w", err)
	}
	return toUserResponse(u), nil
}

// GetProfile returns the user's profile.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user service: get profile: %w", err)
	}
	return &ProfileResponse{
		DisplayName:    p.DisplayName,
		Bio:            p.Bio,
		NativeLanguage: p.NativeLanguage,
		JLPTGoal:       p.JLPTGoal,
	}, nil
}

// UpdateProfile updates the user's profile fields.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) error {
	if err := s.repo.UpsertProfile(ctx, userID, req); err != nil {
		return fmt.Errorf("user service: update profile: %w", err)
	}
	return nil
}

// GetSettings returns the user's app settings.
func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) (*SettingsResponse, error) {
	st, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user service: get settings: %w", err)
	}
	return &SettingsResponse{
		DailyXPGoal:        st.DailyXPGoal,
		EmailReminders:     st.EmailReminders,
		ReminderTime:       st.ReminderTime,
		SRSCardsPerSession: st.SRSCardsPerSession,
	}, nil
}

// UpdateSettings saves the user's app settings.
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, req UpdateSettingsRequest) error {
	if err := s.repo.UpsertSettings(ctx, userID, req); err != nil {
		return fmt.Errorf("user service: update settings: %w", err)
	}
	return nil
}

// ChangePassword verifies the current password and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user service: change password — get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return fmt.Errorf("user service: change password — current password incorrect")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("user service: change password — hash: %w", err)
	}

	return s.repo.UpdatePasswordHash(ctx, userID, string(newHash))
}

// toUserResponse converts a DB User model to the public response DTO.
// This is the single place where PasswordHash is dropped.
func toUserResponse(u *User) *UserResponse {
	return &UserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		Username:      u.Username,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		AvatarURL:     u.AvatarURL,
	}
}
