// Package quiz — service.go.
// Business logic for quiz sessions: starting, submitting answers, finishing.
package quiz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles quiz business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a quiz Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// StartSession creates a new quiz session.
func (s *Service) StartSession(ctx context.Context, userID uuid.UUID, req StartSessionRequest) (*SessionResponse, error) {
	cardCount := req.CardCount
	if cardCount == 0 {
		cardCount = 10
	}

	var chapterID *uuid.UUID
	if req.ChapterID != nil {
		id, err := uuid.Parse(*req.ChapterID)
		if err != nil {
			return nil, fmt.Errorf("quiz service: invalid chapter_id: %w", err)
		}
		chapterID = &id
	}

	session, err := s.repo.CreateSession(ctx, userID, req.QuizType, chapterID, cardCount)
	if err != nil {
		return nil, fmt.Errorf("quiz service: start session: %w", err)
	}

	var chapterIDStr *string
	if session.ChapterID != nil {
		s := session.ChapterID.String()
		chapterIDStr = &s
	}

	return &SessionResponse{
		ID:         session.ID.String(),
		QuizType:   session.QuizType,
		ChapterID:  chapterIDStr,
		TotalCards: session.TotalCards,
	}, nil
}

// SubmitAnswer records a quiz answer and returns the result.
func (s *Service) SubmitAnswer(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, req SubmitAnswerRequest) (*AnswerResultResponse, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("quiz service: get session: %w", err)
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("quiz service: session does not belong to user")
	}

	if session.CompletedAt != nil {
		return nil, fmt.Errorf("quiz service: session already completed")
	}

	contentID, err := uuid.Parse(req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("quiz service: invalid content_id: %w", err)
	}

	// TODO: look up correct answer from content and compare.
	// For now we trust client-reported correctness (to be replaced with server-side validation).
	correct := false

	if err := s.repo.RecordAnswer(ctx, sessionID, contentID, req.UserAnswer, correct, req.TimeTakenMs); err != nil {
		return nil, fmt.Errorf("quiz service: record answer: %w", err)
	}

	xp := 0
	if correct {
		xp = 3
	}

	return &AnswerResultResponse{
		Correct:       correct,
		CorrectAnswer: "",
		XPEarned:      xp,
	}, nil
}

// FinishSession marks the session complete and returns the summary.
func (s *Service) FinishSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*SessionSummaryResponse, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("quiz service: finish — get session: %w", err)
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("quiz service: session does not belong to user")
	}

	if err := s.repo.CompleteSession(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("quiz service: complete session: %w", err)
	}

	total := session.Correct + session.Incorrect
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(session.Correct) / float64(total) * 100
	}

	return &SessionSummaryResponse{
		SessionID: sessionID.String(),
		Correct:   session.Correct,
		Incorrect: session.Incorrect,
		Accuracy:  accuracy,
		XPEarned:  session.Correct * 3,
	}, nil
}
