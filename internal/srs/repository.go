// Package srs — repository.go.
// Owns all SQL for SRS cards, review logs, and deck settings.
package srs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// Repository handles SRS-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs an SRS Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetDueCards returns cards due for review for the given user, up to limit.
func (r *Repository) GetDueCards(ctx context.Context, userID uuid.UUID, limit int) ([]*Card, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, card_type, card_face, content_id, fsrs_state,
		        due, stability, difficulty, elapsed_days, scheduled_days,
		        reps, lapses, last_review, created_at
		 FROM srs_cards
		 WHERE user_id = $1 AND due <= NOW()
		 ORDER BY due ASC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("srs repository: get due cards: %w", err)
	}
	defer rows.Close()

	return scanCards(rows)
}

// GetCardByID fetches a single SRS card by its primary key.
func (r *Repository) GetCardByID(ctx context.Context, id uuid.UUID) (*Card, error) {
	c := &Card{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, card_type, card_face, content_id, fsrs_state,
		        due, stability, difficulty, elapsed_days, scheduled_days,
		        reps, lapses, last_review, created_at
		 FROM srs_cards WHERE id = $1`, id,
	).Scan(&c.ID, &c.UserID, &c.CardType, &c.CardFace, &c.ContentID,
		&c.FSRSState, &c.Due, &c.Stability, &c.Difficulty,
		&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.LastReview, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("srs repository: get card by id: %w", err)
	}
	return c, nil
}

// UpdateCard persists the updated FSRS state after a review.
func (r *Repository) UpdateCard(ctx context.Context, card *Card) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE srs_cards SET
		     fsrs_state = $1, due = $2, stability = $3, difficulty = $4,
		     elapsed_days = $5, scheduled_days = $6, reps = $7, lapses = $8,
		     last_review = $9
		 WHERE id = $10`,
		card.FSRSState, card.Due, card.Stability, card.Difficulty,
		card.ElapsedDays, card.ScheduledDays, card.Reps, card.Lapses,
		card.LastReview, card.ID,
	)
	if err != nil {
		return fmt.Errorf("srs repository: update card: %w", err)
	}
	return nil
}

// CreateReviewLog inserts a review log entry.
func (r *Repository) CreateReviewLog(ctx context.Context, log *ReviewLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO srs_review_log (id, card_id, user_id, rating, reviewed_at, scheduled_at, elapsed_days)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		log.ID, log.CardID, log.UserID, log.Rating,
		log.ReviewedAt, log.ScheduledAt, log.ElapsedDays,
	)
	if err != nil {
		return fmt.Errorf("srs repository: create review log: %w", err)
	}
	return nil
}

// GetDueCount returns the count of new and review cards due for a user.
func (r *Repository) GetDueCount(ctx context.Context, userID uuid.UUID) (newCount, reviewCount int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT
		     COUNT(*) FILTER (WHERE fsrs_state = 0),
		     COUNT(*) FILTER (WHERE fsrs_state != 0 AND due <= NOW())
		 FROM srs_cards WHERE user_id = $1`, userID,
	).Scan(&newCount, &reviewCount)
	if err != nil {
		return 0, 0, fmt.Errorf("srs repository: get due count: %w", err)
	}
	return newCount, reviewCount, nil
}

// CreateCard inserts a new SRS card for a user.
func (r *Repository) CreateCard(ctx context.Context, userID uuid.UUID, cardType, cardFace string, contentID uuid.UUID) (*Card, error) {
	card := &Card{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO srs_cards (id, user_id, card_type, card_face, content_id, fsrs_state, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), 0, 0, 0, 0, 0, 0)
		 RETURNING id, user_id, card_type, card_face, content_id, fsrs_state, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, last_review, created_at`,
		uuid.New(), userID, cardType, cardFace, contentID, fsrs.New,
	).Scan(&card.ID, &card.UserID, &card.CardType, &card.CardFace, &card.ContentID,
		&card.FSRSState, &card.Due, &card.Stability, &card.Difficulty,
		&card.ElapsedDays, &card.ScheduledDays, &card.Reps, &card.Lapses, &card.LastReview, &card.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("srs repository: create card: %w", err)
	}
	return card, nil
}

func scanCards(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]*Card, error) {
	var cards []*Card
	for rows.Next() {
		c := &Card{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.CardType, &c.CardFace, &c.ContentID,
			&c.FSRSState, &c.Due, &c.Stability, &c.Difficulty,
			&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.LastReview, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("srs repository: scan card: %w", err)
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

// unused import guard
var _ = time.Now
