package learn

import (
	"math"
	"testing"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

const ms = time.Millisecond

// A baseline with this many answers gives the user's own median almost all
// the weight in the cold-start blend.
const settledCount = 1_000_000

func settledBaseline(median time.Duration, mad float64) Baseline {
	return Baseline{MedianLogMs: math.Log(float64(median / ms)), MADLogMs: mad, Count: settledCount}
}

func mcq(active time.Duration) GradeInput {
	return GradeInput{QuestionType: common.QuestionKanaMCQRomaji, Correct: true, Attempts: 1, Duration: active}
}

func typed(qt common.QuestionType, active time.Duration, chars int) GradeInput {
	return GradeInput{QuestionType: qt, Correct: true, Attempts: 1, Duration: active, AnswerChars: chars}
}

func with(in GradeInput, edit func(*GradeInput)) GradeInput {
	edit(&in)
	return in
}

// Cold start (no answers) uses the type default as the median and
// DEFAULT_LOG_MAD as the spread, so with the default 300ms per typed
// character: kana_typed (1.5s usual) is fast below ~961ms of thinking time
// and slow above ~2923ms; kana MCQ (2s usual) is fast below ~1282ms and slow
// above ~3898ms.
func TestGrade(t *testing.T) {
	kanaTyped := common.QuestionKanaTyped
	meaningTyped := common.QuestionKanjiMeaningTyped
	readingTyped := common.QuestionKanjiReadingTyped

	tests := []struct {
		name       string
		in         GradeInput
		baseline   Baseline
		wantRating srs.Rating
		wantReason common.GradeReason
	}{
		// Correctness comes first.
		{name: "wrong answer is again", in: with(mcq(2*time.Second), func(in *GradeInput) { in.Correct = false }),
			wantRating: srs.RatingAgain, wantReason: common.GradeReasonWrong},
		{name: "revealed answer is again even if marked correct", in: with(mcq(2*time.Second), func(in *GradeInput) { in.Revealed = true }),
			wantRating: srs.RatingAgain, wantReason: common.GradeReasonRevealed},
		{name: "second attempt is again", in: with(mcq(2*time.Second), func(in *GradeInput) { in.Attempts = 2 }),
			wantRating: srs.RatingAgain, wantReason: common.GradeReasonWrong},
		{name: "wrong fast answer is still again", in: with(mcq(100*ms), func(in *GradeInput) { in.Correct = false }),
			wantRating: srs.RatingAgain, wantReason: common.GradeReasonWrong},
		{name: "on kun mix-up fixed on the second attempt is hard",
			in:         with(typed(readingTyped, 1500*ms, 2), func(in *GradeInput) { in.Attempts = 2; in.ReadingMixUp = true }),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonWrong},

		// Time bands on a type that allows Easy.
		{name: "fast typed answer is easy", in: typed(kanaTyped, 1400*ms, 2),
			wantRating: srs.RatingEasy, wantReason: common.GradeReasonFast},
		{name: "usual typed answer is good", in: typed(kanaTyped, 2100*ms, 2),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "slow typed answer is hard", in: typed(kanaTyped, 4000*ms, 2),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonSlow},
		{name: "typing time is subtracted so a long answer is not slow", in: typed(meaningTyped, 5*time.Second, 10),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "user typing speed replaces the default per character", in: typed(meaningTyped, 3*time.Second, 10),
			baseline: Baseline{TypingPerChar: 100 * ms}, wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "default typing speed makes the same answer fast", in: typed(meaningTyped, 3*time.Second, 10),
			wantRating: srs.RatingEasy, wantReason: common.GradeReasonFast},

		// Multiple choice is capped at Good.
		{name: "fast mcq is capped at good", in: mcq(1000 * ms),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonMCQCap},
		{name: "usual mcq is good", in: mcq(2 * time.Second),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "slow mcq is hard", in: mcq(5 * time.Second),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonSlow},
		{name: "fast kanji meaning mcq is capped at good",
			in:         GradeInput{QuestionType: common.QuestionKanjiMeaningMCQ, Correct: true, Attempts: 1, Duration: 1500 * ms},
			wantRating: srs.RatingGood, wantReason: common.GradeReasonMCQCap},

		// Implausible or interrupted times count as Good.
		{name: "too fast answer is good", in: typed(kanaTyped, 200*ms, 1),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonTooFast},
		{name: "hidden tab is good", in: with(typed(kanaTyped, 1400*ms, 2), func(in *GradeInput) { in.Duration = 5 * time.Second; in.Idle = 3600 * ms }),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonIdle},
		{name: "answer over the active limit is good", in: typed(kanaTyped, 31*time.Second, 2),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonIdle},
		{name: "too fast answer with a hint is hard", in: with(typed(kanaTyped, 200*ms, 1), func(in *GradeInput) { in.HintsUsed = 1 }),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonHint},

		// Caps.
		{name: "hint caps a fast typed answer at hard", in: with(typed(kanaTyped, 1400*ms, 2), func(in *GradeInput) { in.HintsUsed = 1 }),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonHint},
		{name: "accepted typo caps a fast answer at good", in: with(typed(meaningTyped, 2*time.Second, 5), func(in *GradeInput) { in.TypoDistance = 1 }),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonTypo},
		{name: "slow answer with a typo stays hard for being slow",
			in:         with(typed(meaningTyped, 8*time.Second, 5), func(in *GradeInput) { in.TypoDistance = 1 }),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonSlow},
		{name: "one audio replay is free",
			in:         GradeInput{QuestionType: common.QuestionKanaListen, Correct: true, Attempts: 1, Duration: 2 * time.Second, AudioReplays: 1},
			wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "two audio replays cap at hard",
			in:         GradeInput{QuestionType: common.QuestionKanjiListen, Correct: true, Attempts: 1, Duration: 2 * time.Second, AudioReplays: 2},
			wantRating: srs.RatingHard, wantReason: common.GradeReasonReplay},
		{name: "audio time is not counted as answer time",
			in:         GradeInput{QuestionType: common.QuestionKanaListen, Correct: true, Attempts: 1, Duration: 6 * time.Second, Audio: 4 * time.Second},
			wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "build with no wrong tiles is good",
			in:         GradeInput{QuestionType: common.QuestionKanaBuild, Correct: true, Attempts: 1, Duration: 3 * time.Second, Tiles: 3},
			wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "build with one wrong tile is hard",
			in:         GradeInput{QuestionType: common.QuestionKanaBuild, Correct: true, Attempts: 1, Duration: 3 * time.Second, Tiles: 3, WrongTiles: 1},
			wantRating: srs.RatingHard, wantReason: common.GradeReasonWrong},
		{name: "build with two wrong tiles is again",
			in:         GradeInput{QuestionType: common.QuestionKanaBuild, Correct: true, Attempts: 1, Duration: 3 * time.Second, Tiles: 3, WrongTiles: 2},
			wantRating: srs.RatingAgain, wantReason: common.GradeReasonWrong},
		{name: "build with two wrong tiles is again even when too fast",
			in:         GradeInput{QuestionType: common.QuestionKanaBuild, Correct: true, Attempts: 1, Duration: 100 * ms, Tiles: 3, WrongTiles: 2},
			wantRating: srs.RatingAgain, wantReason: common.GradeReasonWrong},
		{name: "few stroke errors cap at hard",
			in:         with(typed(kanaTyped, 1400*ms, 2), func(in *GradeInput) { in.StrokeErrors = 1; in.StrokeCount = 10 }),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonStrokeErrors},
		{name: "stroke error ratio at the limit is again",
			in:         with(typed(kanaTyped, 1400*ms, 2), func(in *GradeInput) { in.StrokeErrors = 1; in.StrokeCount = 4 }),
			wantRating: srs.RatingAgain, wantReason: common.GradeReasonStrokeErrors},
		{name: "full guide caps at hard",
			in:         with(typed(kanaTyped, 1400*ms, 2), func(in *GradeInput) { in.GuideLevel = common.GuideLevelFull }),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonHint},
		{name: "faded guide caps at good",
			in:         with(typed(kanaTyped, 1400*ms, 2), func(in *GradeInput) { in.GuideLevel = common.GuideLevelFaded }),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonHint},
		{name: "no guide does not cap",
			in:         with(typed(kanaTyped, 1400*ms, 2), func(in *GradeInput) { in.GuideLevel = common.GuideLevelNone }),
			wantRating: srs.RatingEasy, wantReason: common.GradeReasonFast},
		{name: "faded guide on mcq names the guide, not the type cap",
			in:         with(mcq(1000*ms), func(in *GradeInput) { in.GuideLevel = common.GuideLevelFaded }),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonHint},
		{name: "lowest of several caps wins",
			in:         with(typed(meaningTyped, 2*time.Second, 5), func(in *GradeInput) { in.TypoDistance = 1; in.HintsUsed = 1 }),
			wantRating: srs.RatingHard, wantReason: common.GradeReasonHint},

		// Baseline and cold-start blend.
		{name: "usual time for a new user is good", in: mcq(2 * time.Second),
			wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "same time is slow for a user who is usually twice as fast", in: mcq(2 * time.Second),
			baseline: settledBaseline(1000*ms, 0.3), wantRating: srs.RatingHard, wantReason: common.GradeReasonSlow},
		{name: "few answers keep the user close to the type default", in: mcq(2 * time.Second),
			baseline: Baseline{MedianLogMs: math.Log(1000), MADLogMs: 0.3, Count: 3}, wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "fast for a usually slow user is easy", in: typed(kanaTyped, 2600*ms, 2),
			baseline: settledBaseline(4000*ms, 0.3), wantRating: srs.RatingEasy, wantReason: common.GradeReasonFast},
		{name: "zero spread is floored so a small delay stays normal", in: mcq(2300 * ms),
			baseline: settledBaseline(2000*ms, 0), wantRating: srs.RatingGood, wantReason: common.GradeReasonNormal},
		{name: "zero spread is floored but a clear delay is slow", in: mcq(2600 * ms),
			baseline: settledBaseline(2000*ms, 0), wantRating: srs.RatingHard, wantReason: common.GradeReasonSlow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Grade(&tt.in, tt.baseline)
			if !got.Rated {
				t.Fatal("rated = false, want true")
			}
			if got.Rating != tt.wantRating || got.Reason != tt.wantReason {
				t.Fatalf("got %d/%s, want %d/%s", got.Rating, got.Reason, tt.wantRating, tt.wantReason)
			}
		})
	}
}

func TestGrade_AnalyticsOnly(t *testing.T) {
	tests := []struct {
		name string
		in   GradeInput
	}{
		{name: "speed round returns no rating", in: GradeInput{QuestionType: common.QuestionKanaSpeed, Correct: true, Attempts: 1, Duration: time.Second}},
		{name: "match grid miss returns no rating", in: GradeInput{QuestionType: common.QuestionKanaMatch, Attempts: 1, Duration: time.Second}},
		{name: "tracing returns no rating", in: GradeInput{QuestionType: common.QuestionKanaTrace, Correct: true, StrokeErrors: 3, StrokeCount: 3}},
		{name: "unknown question type returns no rating", in: GradeInput{QuestionType: "kana_unknown", Correct: true, Duration: time.Second}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Grade(&tt.in, Baseline{})
			if got.Rated || got.Rating != 0 || got.Reason != common.GradeReasonAnalyticsOnly {
				t.Fatalf("got %+v, want unrated analytics_only", got)
			}
		})
	}
}

func TestGrade_Times(t *testing.T) {
	tests := []struct {
		name       string
		in         GradeInput
		baseline   Baseline
		wantActive time.Duration
		wantUsual  time.Duration
	}{
		{name: "active time excludes idle and audio",
			in:         GradeInput{QuestionType: common.QuestionKanaListen, Correct: true, Duration: 6 * time.Second, Idle: time.Second, Audio: 2 * time.Second},
			wantActive: 3 * time.Second, wantUsual: 2 * time.Second},
		{name: "active time never goes negative",
			in:         GradeInput{QuestionType: common.QuestionKanaListen, Correct: true, Duration: time.Second, Audio: 2 * time.Second},
			wantActive: 0, wantUsual: 2 * time.Second},
		{name: "usual time for typed answers adds default typing time",
			in:         GradeInput{QuestionType: common.QuestionKanaTyped, Correct: true, Duration: time.Second, AnswerChars: 2},
			wantActive: time.Second, wantUsual: 2100 * ms},
		{name: "usual time for build scales with tiles",
			in:         GradeInput{QuestionType: common.QuestionKanaBuild, Correct: true, Duration: time.Second, Tiles: 4},
			wantActive: time.Second, wantUsual: 4 * time.Second},
		{name: "usual time blends user and default halfway at the cold-start count",
			in:         mcq(time.Second),
			baseline:   Baseline{MedianLogMs: math.Log(1000), MADLogMs: 0.3, Count: 20},
			wantActive: time.Second, wantUsual: 1414 * ms},
		{name: "analytics types still report active time",
			in:         GradeInput{QuestionType: common.QuestionKanaSpeed, Duration: 900 * ms, Idle: 100 * ms},
			wantActive: 800 * ms, wantUsual: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Grade(&tt.in, tt.baseline)
			if got.Active != tt.wantActive {
				t.Errorf("active = %v, want %v", got.Active, tt.wantActive)
			}
			if (got.Usual - tt.wantUsual).Abs() > ms {
				t.Errorf("usual = %v, want %v", got.Usual, tt.wantUsual)
			}
		})
	}
}

func TestLabelFor(t *testing.T) {
	tests := []struct {
		name   string
		rating srs.Rating
		want   common.GradeLabel
	}{
		{name: "easy is strong", rating: srs.RatingEasy, want: common.GradeLabelStrong},
		{name: "good is good", rating: srs.RatingGood, want: common.GradeLabelGood},
		{name: "hard is shaky", rating: srs.RatingHard, want: common.GradeLabelShaky},
		{name: "again is missed", rating: srs.RatingAgain, want: common.GradeLabelMissed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LabelFor(tt.rating); got != tt.want {
				t.Fatalf("LabelFor(%d) = %s, want %s", tt.rating, got, tt.want)
			}
		})
	}
}

func TestThinkingTime(t *testing.T) {
	tests := []struct {
		name    string
		active  time.Duration
		chars   int
		perChar time.Duration
		want    time.Duration
	}{
		{name: "subtracts typing time", active: 3 * time.Second, chars: 5, perChar: 200 * ms, want: 2 * time.Second},
		{name: "no characters leaves time unchanged", active: 3 * time.Second, chars: 0, perChar: 200 * ms, want: 3 * time.Second},
		{name: "floors at the minimum plausible answer", active: time.Second, chars: 10, perChar: 200 * ms, want: common.MIN_PLAUSIBLE_ANSWER},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ThinkingTime(tt.active, tt.chars, tt.perChar); got != tt.want {
				t.Fatalf("ThinkingTime = %v, want %v", got, tt.want)
			}
		})
	}
}
