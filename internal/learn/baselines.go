package learn

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

// loadBaselines reads the user's recent correct answers once per question
// type in the batch.
func loadBaselines(ctx context.Context, repo BaselineRepo, userID uuid.UUID, types []common.QuestionType) (map[common.QuestionType]Baseline, error) {
	out := make(map[common.QuestionType]Baseline, len(types))
	for _, qt := range types {
		if _, done := out[qt]; done {
			continue
		}
		samples, err := repo.ListBaselineSamples(ctx, userID, qt, common.BASELINE_WINDOW)
		if err != nil {
			return nil, fmt.Errorf("load baselines: %w", err)
		}
		out[qt] = baselineOf(common.QUESTION_TYPE_RULES[qt].Typed, samples)
	}
	return out, nil
}

// Typed answers are scored on thinking time, so their baseline is built the
// same way (learn.BaselineFrom expects that).
func baselineOf(typed bool, samples []BaselineSample) Baseline {
	durations := make([]time.Duration, 0, len(samples))
	if !typed {
		for _, s := range samples {
			durations = append(durations, msDuration(s.ActiveMs))
		}
		return BaselineFrom(durations)
	}
	typing := make([]TypingSample, 0, len(samples))
	for _, s := range samples {
		typing = append(typing, TypingSample{
			Active:     msDuration(s.ActiveMs),
			FirstInput: msDuration(derefInt(s.FirstInputMs)),
			Chars:      answerChars(s.AnswerText),
		})
	}
	perChar := TypingPerChar(typing)
	use := perChar
	if use == 0 {
		use = common.DEFAULT_TYPING_PER_CHAR
	}
	for _, t := range typing {
		durations = append(durations, ThinkingTime(t.Active, t.Chars, use))
	}
	b := BaselineFrom(durations)
	b.TypingPerChar = perChar
	return b
}

func usualTime(qt common.QuestionType, chars, tiles int, b Baseline) time.Duration {
	return Grade(&GradeInput{QuestionType: qt, Correct: true, AnswerChars: chars, Tiles: tiles}, b).Usual
}

// Unrated answers (analytics-only types) have no rating; they read as Good
// or Missed from their correctness.
func labelOf(rating *srs.Rating, correct bool) common.GradeLabel {
	switch {
	case rating != nil:
		return LabelFor(*rating)
	case correct:
		return common.GradeLabelGood
	default:
		return common.GradeLabelMissed
	}
}

func answerChars(text *string) int {
	if text == nil {
		return 0
	}
	return utf8.RuneCountInString(*text)
}

func msDuration(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
