// Package chapter — model.go.
// DB models, request DTOs, and response DTOs for the chapter domain.
package chapter

import (
	"time"

	"github.com/google/uuid"
)

// Chapter is the DB model for the chapters table.
type Chapter struct {
	ID          uuid.UUID `db:"id"`
	Slug        string    `db:"slug"`
	Title       string    `db:"title"`
	JLPTLevel   string    `db:"jlpt_level"`
	OrderIndex  int       `db:"order_index"`
	Description *string   `db:"description"`
	Published   bool      `db:"published"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Concept is the DB model for the concepts table.
type Concept struct {
	ID          uuid.UUID `db:"id"`
	ChapterID   uuid.UUID `db:"chapter_id"`
	Title       string    `db:"title"`
	Explanation string    `db:"explanation"`
	OrderIndex  int       `db:"order_index"`
}

// CulturalInsight is the DB model for the cultural_insights table.
type CulturalInsight struct {
	ID        uuid.UUID `db:"id"`
	ChapterID uuid.UUID `db:"chapter_id"`
	Title     string    `db:"title"`
	Body      string    `db:"body"`
}

// --- Request DTOs ---

// ListChaptersRequest carries query params for listing chapters.
type ListChaptersRequest struct {
	JLPTLevel string `schema:"level" validate:"omitempty,oneof=N5 N4 N3 N2 N1"`
	Page      int    `schema:"page"`
	PerPage   int    `schema:"per_page"`
}

// --- Response DTOs ---

// ChapterResponse is the public representation of a chapter.
type ChapterResponse struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	JLPTLevel   string  `json:"jlpt_level"`
	OrderIndex  int     `json:"order_index"`
	Description *string `json:"description"`
}

// ChapterDetailResponse includes the full chapter with concepts and cultural notes.
type ChapterDetailResponse struct {
	ChapterResponse
	Concepts        []ConceptResponse        `json:"concepts"`
	CulturalInsights []CulturalInsightResponse `json:"cultural_insights"`
}

// ConceptResponse is the public representation of a grammar concept.
type ConceptResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	OrderIndex  int    `json:"order_index"`
}

// CulturalInsightResponse is the public representation of a cultural note.
type CulturalInsightResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}
