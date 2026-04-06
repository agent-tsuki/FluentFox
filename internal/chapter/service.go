// Package chapter — service.go.
// Business logic for chapter retrieval: listing, getting detail, pagination.
package chapter

import (
	"context"
	"fmt"
)

// Service handles chapter business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a chapter Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List returns a paginated list of published chapters, optionally by JLPT level.
func (s *Service) List(ctx context.Context, jlptLevel string, page, perPage int) ([]*ChapterResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	chapters, total, err := s.repo.ListPublished(ctx, jlptLevel, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("chapter service: list: %w", err)
	}

	resp := make([]*ChapterResponse, len(chapters))
	for i, c := range chapters {
		resp[i] = toChapterResponse(c)
	}
	return resp, total, nil
}

// GetDetail returns a chapter with its concepts and cultural insights.
func (s *Service) GetDetail(ctx context.Context, slug string) (*ChapterDetailResponse, error) {
	chapter, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("chapter service: get detail: %w", err)
	}

	concepts, err := s.repo.GetConceptsByChapter(ctx, chapter.ID)
	if err != nil {
		return nil, fmt.Errorf("chapter service: get concepts: %w", err)
	}

	insights, err := s.repo.GetCulturalInsightsByChapter(ctx, chapter.ID)
	if err != nil {
		return nil, fmt.Errorf("chapter service: get cultural insights: %w", err)
	}

	conceptResp := make([]ConceptResponse, len(concepts))
	for i, c := range concepts {
		conceptResp[i] = ConceptResponse{
			ID:          c.ID.String(),
			Title:       c.Title,
			Explanation: c.Explanation,
			OrderIndex:  c.OrderIndex,
		}
	}

	insightResp := make([]CulturalInsightResponse, len(insights))
	for i, ci := range insights {
		insightResp[i] = CulturalInsightResponse{
			ID:    ci.ID.String(),
			Title: ci.Title,
			Body:  ci.Body,
		}
	}

	return &ChapterDetailResponse{
		ChapterResponse:  *toChapterResponse(chapter),
		Concepts:         conceptResp,
		CulturalInsights: insightResp,
	}, nil
}

func toChapterResponse(c *Chapter) *ChapterResponse {
	return &ChapterResponse{
		ID:          c.ID.String(),
		Slug:        c.Slug,
		Title:       c.Title,
		JLPTLevel:   c.JLPTLevel,
		OrderIndex:  c.OrderIndex,
		Description: c.Description,
	}
}
