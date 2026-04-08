package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions/exception"
	"github.com/fluentfox/api/pkg/token"
)

type Service struct {
	repo       *users.Repository
	tokenMaker *token.Maker
}

func AuthService(repo *users.Repository, tokenMaker *token.Maker) *Service {
	return &Service{repo: repo, tokenMaker: tokenMaker}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Check email uniqueness
	exists, err := s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, exception.ErrEmailAlreadyExists
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	// Build user
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: generate uuid: %w", err)
	}

	user := &users.User{
		ID:        id,
		Username:  req.Username,
		Email:     req.Email,
		SecretKey: string(hashed),
		PhoneNo:   req.PhoneNo,
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return s.buildAuthResponse(created)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, exception.ErrAccountInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.SecretKey), []byte(req.Password)); err != nil {
		return nil, exception.ErrInvalidCredentials
	}

	return s.buildAuthResponse(user)
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := toUserResponse(user)
	return &resp, nil
}

// buildAuthResponse generates tokens and builds the response
func (s *Service) buildAuthResponse(user *users.User) (*AuthResponse, error) {
	accessToken, refreshToken, err := s.tokenMaker.GenerateTokenPair(user.ID, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("auth: generate tokens: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserResponse(user),
	}, nil
}
