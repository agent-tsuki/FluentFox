package srs

import (
	"errors"
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

var testNow = time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

func newTestScheduler(t *testing.T) *Scheduler {
	t.Helper()
	s, err := NewScheduler(Params{DesiredRetention: 0.9})
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	return s
}

func timePtr(t time.Time) *time.Time { return &t }

func reviewCard(stability float64, lastReview time.Time) Card {
	return Card{
		State:         StateReview,
		Due:           lastReview.Add(time.Duration(stability) * day),
		Stability:     stability,
		Difficulty:    5,
		Reps:          3,
		ScheduledDays: int(stability),
		LastReview:    timePtr(lastReview),
	}
}

func TestNewScheduler(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		wantErr bool
	}{
		{name: "accepts default weights", params: Params{DesiredRetention: 0.9}},
		{name: "accepts 21 FSRS-6 weights", params: Params{DesiredRetention: 0.85, Weights: defaultWeightsSlice()}},
		{name: "accepts 19 FSRS-5 weights", params: Params{DesiredRetention: 0.9, Weights: defaultWeightsSlice()[:19]}},
		{name: "rejects zero retention", params: Params{DesiredRetention: 0}, wantErr: true},
		{name: "rejects retention above one", params: Params{DesiredRetention: 1.5}, wantErr: true},
		{name: "rejects wrong weight count", params: Params{DesiredRetention: 0.9, Weights: []float64{1, 2, 3}}, wantErr: true},
		{name: "rejects non-finite weights", params: Params{DesiredRetention: 0.9, Weights: nanWeights()}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewScheduler(tt.params)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && s == nil {
				t.Fatal("scheduler is nil")
			}
		})
	}
}

func defaultWeightsSlice() []float64 {
	return []float64{
		0.212, 1.2931, 2.3065, 8.2956, 6.4133, 0.8334, 3.0194, 0.001, 1.8722, 0.1666, 0.796,
		1.4835, 0.0614, 0.2629, 1.6483, 0.6014, 1.8729, 0.5425, 0.0912, 0.0658, 0.1542,
	}
}

func nanWeights() []float64 {
	w := defaultWeightsSlice()
	w[3] = math.NaN()
	return w
}

func TestNewCard(t *testing.T) {
	c := NewCard(testNow)
	if c.State != StateNew || !c.Due.Equal(testNow) || c.Reps != 0 || c.LastReview != nil {
		t.Fatalf("unexpected new card: %+v", c)
	}
}

func TestScheduler_Next(t *testing.T) {
	s := newTestScheduler(t)
	newCard := NewCard(testNow)
	learningLastStep := Card{
		State: StateLearning, Due: testNow, Stability: 2.3, Difficulty: 2.1,
		Reps: 1, LearningStep: 1, LastReview: timePtr(testNow.Add(-10 * time.Minute)),
	}
	relearning := Card{
		State: StateRelearning, Due: testNow, Stability: 1.5, Difficulty: 6,
		Reps: 5, Lapses: 1, LearningStep: 0, LastReview: timePtr(testNow.Add(-10 * time.Minute)),
	}
	review := reviewCard(10, testNow.Add(-10*day))

	tests := []struct {
		name          string
		card          Card
		rating        Rating
		wantState     State
		wantStep      int
		wantDueAfter  time.Duration // checked when non-zero
		wantLapses    int
		wantMinDays   int
		wantGraduated bool
	}{
		{name: "new card rated again starts the first learning step", card: newCard, rating: RatingAgain,
			wantState: StateLearning, wantStep: 0, wantDueAfter: time.Minute},
		{name: "new card rated hard repeats the first step halfway to the second", card: newCard, rating: RatingHard,
			wantState: StateLearning, wantStep: 0, wantDueAfter: 6 * time.Minute},
		{name: "new card rated good moves to the second learning step", card: newCard, rating: RatingGood,
			wantState: StateLearning, wantStep: 1, wantDueAfter: 10 * time.Minute},
		{name: "new card rated easy graduates to review", card: newCard, rating: RatingEasy,
			wantState: StateReview, wantMinDays: 1, wantGraduated: true},
		{name: "last learning step rated good graduates to review", card: learningLastStep, rating: RatingGood,
			wantState: StateReview, wantMinDays: 1, wantGraduated: true},
		{name: "review card rated again lapses into relearning", card: review, rating: RatingAgain,
			wantState: StateRelearning, wantStep: 0, wantDueAfter: 10 * time.Minute, wantLapses: 1},
		{name: "review card rated good stays in review", card: review, rating: RatingGood,
			wantState: StateReview, wantMinDays: 1, wantGraduated: true},
		{name: "relearning card rated good returns to review", card: relearning, rating: RatingGood,
			wantState: StateReview, wantMinDays: 1, wantGraduated: true, wantLapses: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, log, err := s.Next(&tt.card, testNow, tt.rating)
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if got.State != tt.wantState {
				t.Fatalf("state = %d, want %d", got.State, tt.wantState)
			}
			if got.LearningStep != tt.wantStep {
				t.Errorf("learning step = %d, want %d", got.LearningStep, tt.wantStep)
			}
			if tt.wantDueAfter != 0 && !got.Due.Equal(testNow.Add(tt.wantDueAfter)) {
				t.Errorf("due = %v, want now + %v", got.Due, tt.wantDueAfter)
			}
			if tt.wantGraduated && got.ScheduledDays < tt.wantMinDays {
				t.Errorf("scheduled days = %d, want >= %d", got.ScheduledDays, tt.wantMinDays)
			}
			if got.Lapses != tt.wantLapses {
				t.Errorf("lapses = %d, want %d", got.Lapses, tt.wantLapses)
			}
			if got.Reps != tt.card.Reps+1 {
				t.Errorf("reps = %d, want %d", got.Reps, tt.card.Reps+1)
			}
			if got.LastReview == nil || !got.LastReview.Equal(testNow) {
				t.Errorf("last review = %v, want %v", got.LastReview, testNow)
			}
			if log.Rating != tt.rating || log.StateBefore != tt.card.State || !log.ReviewedAt.Equal(testNow) {
				t.Errorf("unexpected log: %+v", log)
			}
		})
	}
}

func TestScheduler_Next_Log(t *testing.T) {
	s := newTestScheduler(t)

	t.Run("new card has no retrievability before and zero elapsed days", func(t *testing.T) {
		c := NewCard(testNow)
		_, log, err := s.Next(&c, testNow, RatingGood)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if log.RetrievabilityBefore != nil || log.ElapsedDays != 0 {
			t.Fatalf("unexpected log: %+v", log)
		}
	})

	t.Run("review card records values before the review", func(t *testing.T) {
		c := reviewCard(10, testNow.Add(-10*day))
		got, log, err := s.Next(&c, testNow, RatingGood)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if log.ElapsedDays != 10 || log.StabilityBefore != 10 || log.DifficultyBefore != 5 {
			t.Fatalf("unexpected log: %+v", log)
		}
		if log.RetrievabilityBefore == nil || math.Abs(*log.RetrievabilityBefore-0.9) > 0.001 {
			t.Fatalf("retrievability before = %v, want about 0.9", log.RetrievabilityBefore)
		}
		if got.Stability <= c.Stability {
			t.Fatalf("stability after good = %v, want above %v", got.Stability, c.Stability)
		}
	})

	t.Run("elapsed days count UTC calendar days", func(t *testing.T) {
		c := reviewCard(3, time.Date(2026, 3, 9, 23, 30, 0, 0, time.UTC))
		_, log, err := s.Next(&c, time.Date(2026, 3, 10, 0, 30, 0, 0, time.UTC), RatingGood)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if log.ElapsedDays != 1 {
			t.Fatalf("elapsed days = %d, want 1", log.ElapsedDays)
		}
	})
}

func TestScheduler_Next_CapsIntervalAtMaximum(t *testing.T) {
	s := newTestScheduler(t)
	c := reviewCard(3000, testNow.Add(-300*day))
	got, _, err := s.Next(&c, testNow, RatingEasy)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	maxDays := int(MaximumInterval / day)
	if got.ScheduledDays > maxDays || got.Due.After(testNow.Add(MaximumInterval)) {
		t.Fatalf("scheduled %d days (due %v), want at most %d", got.ScheduledDays, got.Due, maxDays)
	}
}

func TestScheduler_Next_IsDeterministic(t *testing.T) {
	s := newTestScheduler(t)
	c := reviewCard(20, testNow.Add(-20*day))
	first, _, err := s.Next(&c, testNow, RatingGood)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	second, _, err := s.Next(&c, testNow, RatingGood)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if !first.Due.Equal(second.Due) || first.Stability != second.Stability {
		t.Fatalf("results differ: %+v vs %+v", first, second)
	}
}

func TestScheduler_Next_Errors(t *testing.T) {
	s := newTestScheduler(t)
	tests := []struct {
		name        string
		card        Card
		rating      Rating
		wantInvalid bool
	}{
		{name: "rejects unknown state", card: Card{State: 7, Due: testNow}, rating: RatingGood, wantInvalid: true},
		{name: "rejects negative reps", card: Card{State: StateNew, Due: testNow, Reps: -1}, rating: RatingGood, wantInvalid: true},
		{name: "rejects zero rating", card: NewCard(testNow), rating: 0},
		{name: "rejects rating above easy", card: NewCard(testNow), rating: 5},
		{name: "rejects review card without stability", card: Card{State: StateReview, Due: testNow}, rating: RatingGood},
		{name: "rejects last review in the future", card: reviewCard(5, testNow.Add(day)), rating: RatingGood},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := s.Next(&tt.card, testNow, tt.rating)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if tt.wantInvalid && !errors.Is(err, ErrInvalidCard) {
				t.Fatalf("err = %v, want ErrInvalidCard", err)
			}
		})
	}
}

func TestScheduler_Retrievability(t *testing.T) {
	s := newTestScheduler(t)
	tests := []struct {
		name string
		card Card
		want float64
	}{
		{name: "new card is zero", card: NewCard(testNow), want: 0},
		{name: "just reviewed card is one", card: reviewCard(7, testNow), want: 1},
		{name: "card at its stability equals the 90 percent definition", card: reviewCard(7, testNow.Add(-7*day)), want: 0.9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Retrievability(&tt.card, testNow)
			if err != nil {
				t.Fatalf("Retrievability: %v", err)
			}
			if math.Abs(got-tt.want) > 0.001 {
				t.Fatalf("retrievability = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("decreases as time passes", func(t *testing.T) {
		c := reviewCard(7, testNow)
		early, err := s.Retrievability(&c, testNow.Add(2*day))
		if err != nil {
			t.Fatalf("Retrievability: %v", err)
		}
		late, err := s.Retrievability(&c, testNow.Add(20*day))
		if err != nil {
			t.Fatalf("Retrievability: %v", err)
		}
		if late >= early {
			t.Fatalf("late %v should be below early %v", late, early)
		}
	})

	t.Run("rejects unknown state", func(t *testing.T) {
		c := Card{State: 9}
		if _, err := s.Retrievability(&c, testNow); !errors.Is(err, ErrInvalidCard) {
			t.Fatalf("err = %v, want ErrInvalidCard", err)
		}
	})
}

type fixedRand struct{ n int64 }

func (f fixedRand) Int64N(int64) int64 { return f.n }

func TestScheduler_NewKnownCard(t *testing.T) {
	s := newTestScheduler(t)
	newCard := NewCard(testNow)
	firstGood, _, err := s.Next(&newCard, testNow, RatingGood)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}

	tests := []struct {
		name     string
		rnd      RandSource
		wantDays int
	}{
		{name: "lowest draw is due after one day", rnd: fixedRand{0}, wantDays: 1},
		{name: "highest draw is due after seven days", rnd: fixedRand{6}, wantDays: 7},
		{name: "middle draw is due after four days", rnd: fixedRand{3}, wantDays: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := s.NewKnownCard(testNow, tt.rnd)
			if c.State != StateReview || c.Stability != 7 || c.ScheduledDays != tt.wantDays {
				t.Fatalf("unexpected card: %+v", c)
			}
			if !c.Due.Equal(testNow.Add(time.Duration(tt.wantDays) * day)) {
				t.Fatalf("due = %v, want now + %d days", c.Due, tt.wantDays)
			}
			if c.Difficulty != firstGood.Difficulty {
				t.Fatalf("difficulty = %v, want the first-good default %v", c.Difficulty, firstGood.Difficulty)
			}
			if c.LastReview == nil || !c.LastReview.Equal(testNow) {
				t.Fatalf("last review = %v, want now", c.LastReview)
			}
		})
	}

	t.Run("seeded random source keeps due within one to seven days", func(t *testing.T) {
		rnd := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // seeded on purpose so the test is deterministic
		seen := map[int]bool{}
		for range 200 {
			c := s.NewKnownCard(testNow, rnd)
			if c.ScheduledDays < 1 || c.ScheduledDays > 7 {
				t.Fatalf("scheduled days = %d, want 1 to 7", c.ScheduledDays)
			}
			seen[c.ScheduledDays] = true
		}
		if len(seen) != 7 {
			t.Fatalf("saw %d distinct days, want all 7", len(seen))
		}
	})

	t.Run("known card can be reviewed", func(t *testing.T) {
		c := s.NewKnownCard(testNow, fixedRand{2})
		if _, _, err := s.Next(&c, c.Due, RatingGood); err != nil {
			t.Fatalf("Next on known card: %v", err)
		}
	})
}
