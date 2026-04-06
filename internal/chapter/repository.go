// Package chapter — repository.go.
// Owns all SQL for chapters, concepts, and cultural insights.
package chapter

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles chapter-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a chapter Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListPublished returns published chapters, optionally filtered by JLPT level.
func (r *Repository) ListPublished(ctx context.Context, jlptLevel string, limit, offset int) ([]*Chapter, int, error) {
	query := `SELECT id, slug, title, jlpt_level, order_index, description, published, created_at, updated_at
	          FROM chapters WHERE published = true`
	args := []any{limit, offset}
	paramIdx := 3

	if jlptLevel != "" {
		query += fmt.Sprintf(" AND jlpt_level = $%d", paramIdx)
		args = append(args, jlptLevel)
		paramIdx++
	}
	query += " ORDER BY jlpt_level, order_index LIMIT $1 OFFSET $2"

	var total int
	countQuery := `SELECT COUNT(*) FROM chapters WHERE published = true`
	if jlptLevel != "" {
		countQuery += " AND jlpt_level = $1"
		if err := r.pool.QueryRow(ctx, countQuery, jlptLevel).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("chapter repository: count: %w", err)
		}
	} else {
		if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("chapter repository: count: %w", err)
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("chapter repository: list: %w", err)
	}
	defer rows.Close()

	var chapters []*Chapter
	for rows.Next() {
		c := &Chapter{}
		if err := rows.Scan(&c.ID, &c.Slug, &c.Title, &c.JLPTLevel, &c.OrderIndex,
			&c.Description, &c.Published, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("chapter repository: scan: %w", err)
		}
		chapters = append(chapters, c)
	}
	return chapters, total, rows.Err()
}

// GetBySlug fetches a single chapter by its URL slug.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Chapter, error) {
	c := &Chapter{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, slug, title, jlpt_level, order_index, description, published, created_at, updated_at
		 FROM chapters WHERE slug = $1 AND published = true`, slug,
	).Scan(&c.ID, &c.Slug, &c.Title, &c.JLPTLevel, &c.OrderIndex,
		&c.Description, &c.Published, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("chapter repository: get by slug: %w", err)
	}
	return c, nil
}

// GetConceptsByChapter fetches all concepts for a chapter, ordered.
func (r *Repository) GetConceptsByChapter(ctx context.Context, chapterID uuid.UUID) ([]*Concept, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chapter_id, title, explanation, order_index
		 FROM concepts WHERE chapter_id = $1 ORDER BY order_index`, chapterID)
	if err != nil {
		return nil, fmt.Errorf("chapter repository: get concepts: %w", err)
	}
	defer rows.Close()

	var concepts []*Concept
	for rows.Next() {
		c := &Concept{}
		if err := rows.Scan(&c.ID, &c.ChapterID, &c.Title, &c.Explanation, &c.OrderIndex); err != nil {
			return nil, fmt.Errorf("chapter repository: scan concept: %w", err)
		}
		concepts = append(concepts, c)
	}
	return concepts, rows.Err()
}

// GetCulturalInsightsByChapter fetches cultural notes for a chapter.
func (r *Repository) GetCulturalInsightsByChapter(ctx context.Context, chapterID uuid.UUID) ([]*CulturalInsight, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chapter_id, title, body FROM cultural_insights WHERE chapter_id = $1`, chapterID)
	if err != nil {
		return nil, fmt.Errorf("chapter repository: get cultural insights: %w", err)
	}
	defer rows.Close()

	var insights []*CulturalInsight
	for rows.Next() {
		ci := &CulturalInsight{}
		if err := rows.Scan(&ci.ID, &ci.ChapterID, &ci.Title, &ci.Body); err != nil {
			return nil, fmt.Errorf("chapter repository: scan cultural insight: %w", err)
		}
		insights = append(insights, ci)
	}
	return insights, rows.Err()
}
