// Package srs — model.go.
// DB models, request DTOs, and response DTOs for the spaced repetition system domain.
// The SRS algorithm is go-fsrs/v3. Card scheduling data maps directly to fsrs card state.
package srs

import (
	"time"

	"github.com/google/uuid"
	"github.com/open-spaced-repetition/go-fsrs/v3"
)

// Card is the DB model for the srs_cards table.
// It stores the FSRS scheduling state alongside a reference to the content item.
type Card struct {
	ID            uuid.UUID      `db:"id"`
	UserID        uuid.UUID      `db:"user_id"`
	CardType      string         `db:"card_type"`  // vocabulary, kanji, character, grammar
	CardFace      string         `db:"card_face"`  // recognition, production, meaning
	ContentID     uuid.UUID      `db:"content_id"` // FK to vocab/kanji/character/concept
	FSRSState     fsrs.State     `db:"fsrs_state"`
	Due           time.Time      `db:"due"`
	Stability     float64        `db:"stability"`
	Difficulty    float64        `db:"difficulty"`
	ElapsedDays   uint64         `db:"elapsed_days"`
	ScheduledDays uint64         `db:"scheduled_days"`
	Reps          uint64         `db:"reps"`
	Lapses        uint64         `db:"lapses"`
	LastReview    *time.Time     `db:"last_review"`
	CreatedAt     time.Time      `db:"created_at"`
}

// ReviewLog is the DB model for the srs_review_log table.
type ReviewLog struct {
	ID         uuid.UUID  `db:"id"`
	CardID     uuid.UUID  `db:"card_id"`
	UserID     uuid.UUID  `db:"user_id"`
	Rating     int        `db:"rating"` // 1=Again 2=Hard 3=Good 4=Easy
	ReviewedAt time.Time  `db:"reviewed_at"`
	ScheduledAt time.Time `db:"scheduled_at"`
	ElapsedDays uint64    `db:"elapsed_days"`
}

// DeckSettings is the DB model for the srs_deck_settings table.
type DeckSettings struct {
	UserID           uuid.UUID `db:"user_id"`
	NewCardsPerDay   int       `db:"new_cards_per_day"`
	ReviewsPerDay    int       `db:"reviews_per_day"`
	LeechThreshold   int       `db:"leech_threshold"` // lapses before a card is flagged
}

// --- Request DTOs ---

// SubmitReviewRequest is the payload for POST /srs/review.
type SubmitReviewRequest struct {
	CardID string `json:"card_id" validate:"required,uuid"`
	Rating int    `json:"rating"  validate:"required,min=1,max=4"`
}

// --- Response DTOs ---

// CardResponse is the public representation of an SRS card for study sessions.
type CardResponse struct {
	ID          string     `json:"id"`
	CardType    string     `json:"card_type"`
	CardFace    string     `json:"card_face"`
	ContentID   string     `json:"content_id"`
	Due         time.Time  `json:"due"`
	IsNew       bool       `json:"is_new"`
	Lapses      uint64     `json:"lapses"`
}

// ReviewResultResponse is returned after submitting a review.
type ReviewResultResponse struct {
	CardID      string    `json:"card_id"`
	NextDue     time.Time `json:"next_due"`
	ScheduledDays uint64  `json:"scheduled_days"`
	IsLeech     bool      `json:"is_leech"`
	XPEarned    int       `json:"xp_earned"`
}

// DueCountResponse summarises how many cards are due.
type DueCountResponse struct {
	New      int `json:"new"`
	Review   int `json:"review"`
	Total    int `json:"total"`
}
