// Package srs schedules spaced-repetition cards with FSRS. It wraps go-fsrs so
// the rest of the code works with plain values that match the srs_card columns.
package srs

import (
	"errors"
	"fmt"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
)

type State int8

const (
	StateNew        State = 0
	StateLearning   State = 1
	StateReview     State = 2
	StateRelearning State = 3
)

type Rating int8

const (
	RatingAgain Rating = 1
	RatingHard  Rating = 2
	RatingGood  Rating = 3
	RatingEasy  Rating = 4
)

const (
	MaximumInterval      time.Duration = 365 * day
	KnownCardStability   time.Duration = 7 * day
	KnownCardMinDueAfter time.Duration = 1 * day
	KnownCardMaxDueAfter time.Duration = 7 * day

	day time.Duration = 24 * time.Hour
)

var (
	learningSteps   = []time.Duration{time.Minute, 10 * time.Minute}
	relearningSteps = []time.Duration{10 * time.Minute}
)

var ErrInvalidCard = errors.New("srs: invalid card")

type Card struct {
	State      State
	Due        time.Time
	Stability  float64
	Difficulty float64
	Reps       int
	Lapses     int
	// Index of the current learning or relearning step; go-fsrs tracks the
	// same thing as steps remaining.
	LearningStep  int
	ScheduledDays int
	LastReview    *time.Time
}

// Log holds the card values before the review, for the review_log row. The
// values after the review are on the returned Card.
type Log struct {
	Rating               Rating
	ReviewedAt           time.Time
	StateBefore          State
	StabilityBefore      float64
	DifficultyBefore     float64
	RetrievabilityBefore *float64
	ElapsedDays          int
}

type Params struct {
	DesiredRetention float64
	// Weights are the user's optimised FSRS weights; nil means the defaults.
	// 17, 19 and 21 values are accepted and migrated to FSRS-6.
	Weights []float64
}

type RandSource interface {
	Int64N(n int64) int64
}

type Scheduler struct {
	fsrs *fsrs.FSRS
}

func NewScheduler(p Params) (*Scheduler, error) {
	param := fsrs.DefaultParam()
	param.RequestRetention = p.DesiredRetention
	param.MaximumInterval = durationToDays(MaximumInterval)
	param.EnableFuzz = true
	param.EnableShortTerm = true
	param.LearningSteps = durationsToMinutes(learningSteps)
	param.RelearningSteps = durationsToMinutes(relearningSteps)

	if p.Weights != nil {
		w, err := fsrs.MigrateWeights(p.Weights)
		if err != nil {
			return nil, fmt.Errorf("srs: migrate weights: %w", err)
		}
		param.W = w
	}

	// fsrs.NewFSRS silently falls back to defaults on invalid parameters, so
	// validate first to surface a bad retention or weight set.
	if err := param.Validate(); err != nil {
		return nil, fmt.Errorf("srs: invalid params: %w", err)
	}
	return &Scheduler{fsrs: fsrs.NewFSRS(param)}, nil
}

func NewCard(now time.Time) Card {
	return Card{State: StateNew, Due: now}
}

func (s *Scheduler) Next(card *Card, now time.Time, rating Rating) (Card, Log, error) {
	in, err := s.toFSRS(card)
	if err != nil {
		return Card{}, Log{}, err
	}
	retrievability, err := s.retrievabilityBefore(card, now)
	if err != nil {
		return Card{}, Log{}, err
	}
	info, err := s.fsrs.Next(in, now, fsrs.Rating(rating))
	if err != nil {
		return Card{}, Log{}, fmt.Errorf("srs: next: %w", err)
	}
	log := Log{
		Rating:               rating,
		ReviewedAt:           now,
		StateBefore:          card.State,
		StabilityBefore:      card.Stability,
		DifficultyBefore:     card.Difficulty,
		RetrievabilityBefore: retrievability,
		ElapsedDays:          elapsedDays(card.LastReview, now),
	}
	return s.fromFSRS(&info.Card), log, nil
}

func (s *Scheduler) Retrievability(card *Card, now time.Time) (float64, error) {
	in, err := s.toFSRS(card)
	if err != nil {
		return 0, err
	}
	r, err := s.fsrs.Retrievability(in, now)
	if err != nil {
		return 0, fmt.Errorf("srs: retrievability: %w", err)
	}
	return r, nil
}

// NewKnownCard is used when a skip test proves the item is already known.
// The due date is spread over several days so a passed test does not create
// one large pile of reviews on a single day.
func (s *Scheduler) NewKnownCard(now time.Time, rnd RandSource) Card {
	spread := int64((KnownCardMaxDueAfter - KnownCardMinDueAfter) / day)
	dueAfter := KnownCardMinDueAfter + time.Duration(rnd.Int64N(spread+1))*day
	reviewed := now
	return Card{
		State:         StateReview,
		Due:           now.Add(dueAfter),
		Stability:     durationToDays(KnownCardStability),
		Difficulty:    s.defaultDifficulty(),
		Reps:          1,
		ScheduledDays: int(dueAfter / day),
		LastReview:    &reviewed,
	}
}

// The difficulty FSRS gives a new card on its first Good answer.
func (s *Scheduler) defaultDifficulty() float64 {
	return s.fsrs.NextState(nil, s.fsrs.RequestRetention, 0, fsrs.Good).Memory.Difficulty
}

func (s *Scheduler) retrievabilityBefore(card *Card, now time.Time) (*float64, error) {
	if card.State == StateNew {
		return nil, nil
	}
	r, err := s.Retrievability(card, now)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Scheduler) toFSRS(c *Card) (fsrs.Card, error) {
	if c.State < StateNew || c.State > StateRelearning {
		return fsrs.Card{}, fmt.Errorf("%w: state %d", ErrInvalidCard, c.State)
	}
	if c.Reps < 0 || c.Lapses < 0 || c.ScheduledDays < 0 || c.LearningStep < 0 {
		return fsrs.Card{}, fmt.Errorf("%w: negative counter", ErrInvalidCard)
	}
	out := fsrs.Card{
		Due:            c.Due,
		Stability:      c.Stability,
		Difficulty:     c.Difficulty,
		ScheduledDays:  uint64(c.ScheduledDays),
		Reps:           uint64(c.Reps),
		Lapses:         uint64(c.Lapses),
		State:          fsrs.State(c.State),
		RemainingSteps: max(s.stepCount(c.State)-c.LearningStep, 0),
	}
	if c.LastReview != nil {
		out.LastReview = *c.LastReview
	}
	return out, nil
}

func (s *Scheduler) fromFSRS(c *fsrs.Card) Card {
	state := State(c.State)
	out := Card{
		State:         state,
		Due:           c.Due,
		Stability:     c.Stability,
		Difficulty:    c.Difficulty,
		Reps:          int(c.Reps),          //nolint:gosec // review counts stay far below MaxInt
		Lapses:        int(c.Lapses),        //nolint:gosec // as above
		ScheduledDays: int(c.ScheduledDays), //nolint:gosec // capped by MaximumInterval
	}
	if state == StateLearning || state == StateRelearning {
		out.LearningStep = s.stepCount(state) - c.RemainingSteps
	}
	if !c.LastReview.IsZero() {
		last := c.LastReview
		out.LastReview = &last
	}
	return out
}

func (s *Scheduler) stepCount(state State) int {
	switch state {
	case StateLearning:
		return len(s.fsrs.LearningSteps)
	case StateRelearning:
		return len(s.fsrs.RelearningSteps)
	default:
		return 0
	}
}

// Counts calendar days in UTC, the same way go-fsrs measures elapsed time.
func elapsedDays(lastReview *time.Time, now time.Time) int {
	if lastReview == nil {
		return 0
	}
	from := truncateToUTCDay(*lastReview)
	to := truncateToUTCDay(now)
	if to.Before(from) {
		return 0
	}
	return int(to.Sub(from) / day)
}

func truncateToUTCDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func durationToDays(d time.Duration) float64 {
	return float64(d) / float64(day)
}

func durationsToMinutes(ds []time.Duration) []float64 {
	out := make([]float64, len(ds))
	for i, d := range ds {
		out[i] = d.Minutes()
	}
	return out
}
