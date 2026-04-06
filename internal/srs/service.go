// Package srs — service.go.
// Owns SRS business logic: scheduling reviews using go-fsrs, detecting leeches,
// awarding XP per review. Never knows about HTTP.
package srs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

const (
	leechThreshold = 8  // cards with >= 8 lapses are flagged as leeches
	xpPerCorrect   = 2  // XP per correct (Good/Easy) review
)

// Service handles SRS business logic.
type Service struct {
	repo *Repository
}

// NewService constructs an SRS Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetDueCards returns cards due for the authenticated user.
func (s *Service) GetDueCards(ctx context.Context, userID uuid.UUID, limit int) ([]*CardResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	cards, err := s.repo.GetDueCards(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("srs service: get due cards: %w", err)
	}

	resp := make([]*CardResponse, len(cards))
	for i, c := range cards {
		resp[i] = &CardResponse{
			ID:        c.ID.String(),
			CardType:  c.CardType,
			CardFace:  c.CardFace,
			ContentID: c.ContentID.String(),
			Due:       c.Due,
			IsNew:     c.FSRSState == fsrs.New,
			Lapses:    c.Lapses,
		}
	}
	return resp, nil
}

// SubmitReview processes a review rating, updates the card's FSRS state,
// logs the review, and returns the next due date and whether the card is a leech.
func (s *Service) SubmitReview(ctx context.Context, userID uuid.UUID, req SubmitReviewRequest) (*ReviewResultResponse, error) {
	cardID, err := uuid.Parse(req.CardID)
	if err != nil {
		return nil, fmt.Errorf("srs service: invalid card id: %w", err)
	}

	card, err := s.repo.GetCardByID(ctx, cardID)
	if err != nil {
		return nil, fmt.Errorf("srs service: get card: %w", err)
	}

	if card.UserID != userID {
		return nil, fmt.Errorf("srs service: card does not belong to user")
	}

	scheduler := fsrs.NewFSRS(fsrs.DefaultParam())
	now := time.Now()

	fsrsCard := fsrs.Card{
		Due:           card.Due,
		Stability:     float64(card.Stability),
		Difficulty:    float64(card.Difficulty),
		ElapsedDays:   card.ElapsedDays,
		ScheduledDays: card.ScheduledDays,
		Reps:          card.Reps,
		Lapses:        card.Lapses,
		State:         card.FSRSState,
		LastReview:    now,
	}

	rating := fsrs.Rating(req.Rating)
	schedulingCards := scheduler.Repeat(fsrsCard, now)
	scheduled := schedulingCards[rating].Card

	card.FSRSState = scheduled.State
	card.Due = scheduled.Due
	card.Stability = float64(scheduled.Stability)
	card.Difficulty = float64(scheduled.Difficulty)
	card.ElapsedDays = scheduled.ElapsedDays
	card.ScheduledDays = scheduled.ScheduledDays
	card.Reps = scheduled.Reps
	card.Lapses = scheduled.Lapses
	card.LastReview = &now

	if err := s.repo.UpdateCard(ctx, card); err != nil {
		return nil, fmt.Errorf("srs service: update card: %w", err)
	}

	reviewLog := &ReviewLog{
		ID:           uuid.New(),
		CardID:       card.ID,
		UserID:       userID,
		Rating:       req.Rating,
		ReviewedAt:   now,
		ScheduledAt:  card.Due,
		ElapsedDays:  card.ElapsedDays,
	}
	if err := s.repo.CreateReviewLog(ctx, reviewLog); err != nil {
		return nil, fmt.Errorf("srs service: log review: %w", err)
	}

	isLeech := card.Lapses >= leechThreshold
	xp := 0
	if req.Rating >= 3 { // Good or Easy
		xp = xpPerCorrect
	}

	return &ReviewResultResponse{
		CardID:        card.ID.String(),
		NextDue:       card.Due,
		ScheduledDays: card.ScheduledDays,
		IsLeech:       isLeech,
		XPEarned:      xp,
	}, nil
}

// GetDueCount returns a summary of due cards for the dashboard widget.
func (s *Service) GetDueCount(ctx context.Context, userID uuid.UUID) (*DueCountResponse, error) {
	newCount, reviewCount, err := s.repo.GetDueCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("srs service: get due count: %w", err)
	}
	return &DueCountResponse{
		New:    newCount,
		Review: reviewCount,
		Total:  newCount + reviewCount,
	}, nil
}
