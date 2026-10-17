package learn

import (
	"math"
	"time"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

type GradeInput struct {
	QuestionType common.QuestionType
	Correct      bool
	Revealed     bool
	Attempts     int
	// The first attempt gave the on reading for a kun question (or the
	// reverse) and a later attempt was right. It is graded Hard, not Again.
	ReadingMixUp bool
	HintsUsed    int
	TypoDistance int
	AudioReplays int
	// AnswerChars is the typed answer length; Tiles and WrongTiles are for
	// build questions.
	AnswerChars  int
	Tiles        int
	WrongTiles   int
	StrokeErrors int
	StrokeCount  int
	// Empty when the question showed no guide.
	GuideLevel common.GuideLevel
	Duration   time.Duration
	Idle       time.Duration
	Audio      time.Duration
}

// Baseline describes the user's recent correct answers for one question type,
// in log milliseconds. TypingPerChar is zero until the user has typed answers.
type Baseline struct {
	MedianLogMs   float64
	MADLogMs      float64
	Count         int
	TypingPerChar time.Duration
}

// Rated is false for analytics-only question types, which never change the
// schedule; Rating is then zero.
type GradeResult struct {
	Rating srs.Rating
	Rated  bool
	Reason common.GradeReason
	Active time.Duration
	Usual  time.Duration
}

type capCandidate struct {
	rating srs.Rating
	reason common.GradeReason
}

func Grade(in *GradeInput, b Baseline) GradeResult {
	res := GradeResult{Active: activeTime(in)}
	rule, ok := common.QUESTION_TYPE_RULES[in.QuestionType]
	if !ok || !rule.UpdatesSchedule {
		res.Reason = common.GradeReasonAnalyticsOnly
		return res
	}

	res.Rated = true
	perChar := typingPerChar(b)
	median, mad := blendedLogStats(b, typeUsualTime(rule, in.Tiles))
	res.Usual = msToDuration(math.Exp(median)) + typingTime(rule, in.AnswerChars, perChar)

	if reason, missed := missedReason(in); missed {
		res.Rating, res.Reason = srs.RatingAgain, reason
		return res
	}
	limit := gradeCap(in, rule)
	if limit.rating == srs.RatingAgain {
		res.Rating, res.Reason = limit.rating, limit.reason
		return res
	}

	timed := timeGrade(in, res.Active, ThinkingTime(res.Active, typedChars(rule, in), perChar), median, mad)
	if limit.rating < timed.rating {
		timed = limit
	}
	res.Rating, res.Reason = timed.rating, timed.reason
	return res
}

func LabelFor(r srs.Rating) common.GradeLabel {
	switch r {
	case srs.RatingEasy:
		return common.GradeLabelStrong
	case srs.RatingGood:
		return common.GradeLabelGood
	case srs.RatingHard:
		return common.GradeLabelShaky
	default:
		return common.GradeLabelMissed
	}
}

// ThinkingTime removes the time spent typing so long answers are not graded
// as slow. It never drops below MIN_PLAUSIBLE_ANSWER so the log stays finite.
func ThinkingTime(active time.Duration, chars int, perChar time.Duration) time.Duration {
	return max(active-time.Duration(chars)*perChar, common.MIN_PLAUSIBLE_ANSWER)
}

func missedReason(in *GradeInput) (common.GradeReason, bool) {
	switch {
	case in.Revealed:
		return common.GradeReasonRevealed, true
	case !in.Correct:
		return common.GradeReasonWrong, true
	case in.Attempts > 1 && !in.ReadingMixUp:
		return common.GradeReasonWrong, true
	default:
		return "", false
	}
}

// On equal caps the earlier rule wins. The type maximum is checked last so it
// is only named when nothing more specific lowered the grade.
var capRules = []func(*GradeInput) (capCandidate, bool){
	buildTilesCap,
	strokeErrorsCap,
	hintCap,
	guideCap,
	readingMixUpCap,
	replayCap,
	typoCap,
}

func gradeCap(in *GradeInput, rule common.QuestionTypeRule) capCandidate {
	limit := capCandidate{rating: srs.RatingEasy, reason: common.GradeReasonNormal}
	for _, capRule := range capRules {
		if c, ok := capRule(in); ok && c.rating < limit.rating {
			limit = c
		}
	}
	if rule.MaxRating < limit.rating {
		limit = capCandidate{rule.MaxRating, common.GradeReasonMCQCap}
	}
	return limit
}

func buildTilesCap(in *GradeInput) (capCandidate, bool) {
	switch {
	case in.WrongTiles >= common.BUILD_TILES_AGAIN:
		return capCandidate{srs.RatingAgain, common.GradeReasonWrong}, true
	case in.WrongTiles > 0:
		return capCandidate{srs.RatingHard, common.GradeReasonWrong}, true
	default:
		return capCandidate{}, false
	}
}

func strokeErrorsCap(in *GradeInput) (capCandidate, bool) {
	if in.StrokeErrors <= 0 || in.StrokeCount <= 0 {
		return capCandidate{}, false
	}
	if float64(in.StrokeErrors)/float64(in.StrokeCount) >= common.STROKE_ERROR_RATIO_AGAIN {
		return capCandidate{srs.RatingAgain, common.GradeReasonStrokeErrors}, true
	}
	return capCandidate{srs.RatingHard, common.GradeReasonStrokeErrors}, true
}

func hintCap(in *GradeInput) (capCandidate, bool) {
	return capCandidate{srs.RatingHard, common.GradeReasonHint}, in.HintsUsed > 0
}

// A full guide is close to being shown the answer; a faded one still helps.
func guideCap(in *GradeInput) (capCandidate, bool) {
	switch in.GuideLevel {
	case common.GuideLevelFull:
		return capCandidate{srs.RatingHard, common.GradeReasonHint}, true
	case common.GuideLevelFaded:
		return capCandidate{srs.RatingGood, common.GradeReasonHint}, true
	default:
		return capCandidate{}, false
	}
}

func readingMixUpCap(in *GradeInput) (capCandidate, bool) {
	return capCandidate{srs.RatingHard, common.GradeReasonWrong}, in.ReadingMixUp
}

func replayCap(in *GradeInput) (capCandidate, bool) {
	return capCandidate{srs.RatingHard, common.GradeReasonReplay}, in.AudioReplays > common.FREE_AUDIO_REPLAYS
}

func typoCap(in *GradeInput) (capCandidate, bool) {
	return capCandidate{srs.RatingGood, common.GradeReasonTypo}, in.TypoDistance > 0
}

// Hidden tabs and implausible times say nothing about recall speed, so they
// neither reward nor punish: the answer counts as Good.
func timeGrade(in *GradeInput, active, thinking time.Duration, median, mad float64) capCandidate {
	switch {
	case in.Idle > 0 || active > common.MAX_ACTIVE_ANSWER:
		return capCandidate{srs.RatingGood, common.GradeReasonIdle}
	case active < common.MIN_PLAUSIBLE_ANSWER:
		return capCandidate{srs.RatingGood, common.GradeReasonTooFast}
	}
	z := (math.Log(durationToMs(thinking)) - median) / (common.MAD_TO_SIGMA * mad)
	switch {
	case z > common.SLOW_Z:
		return capCandidate{srs.RatingHard, common.GradeReasonSlow}
	case z < common.FAST_Z:
		return capCandidate{srs.RatingEasy, common.GradeReasonFast}
	default:
		return capCandidate{srs.RatingGood, common.GradeReasonNormal}
	}
}

// New users have few answers, so their own median is blended with the type
// default and takes over as answers accumulate.
func blendedLogStats(b Baseline, usual time.Duration) (median, mad float64) {
	w := float64(b.Count) / (float64(b.Count) + common.BASELINE_COLD_START)
	median = w*b.MedianLogMs + (1-w)*math.Log(durationToMs(usual))
	mad = w*b.MADLogMs + (1-w)*common.DEFAULT_LOG_MAD
	return median, max(mad, common.MIN_LOG_MAD)
}

func typeUsualTime(rule common.QuestionTypeRule, tiles int) time.Duration {
	return max(rule.UsualTime+time.Duration(tiles)*rule.UsualTimePerTile, common.MIN_PLAUSIBLE_ANSWER)
}

func typingPerChar(b Baseline) time.Duration {
	if b.TypingPerChar > 0 {
		return b.TypingPerChar
	}
	return common.DEFAULT_TYPING_PER_CHAR
}

func typedChars(rule common.QuestionTypeRule, in *GradeInput) int {
	if !rule.Typed {
		return 0
	}
	return in.AnswerChars
}

func typingTime(rule common.QuestionTypeRule, chars int, perChar time.Duration) time.Duration {
	if !rule.Typed {
		return 0
	}
	return time.Duration(chars) * perChar
}

func activeTime(in *GradeInput) time.Duration {
	return max(in.Duration-in.Idle-in.Audio, 0)
}

func durationToMs(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func msToDuration(ms float64) time.Duration {
	return time.Duration(math.Round(ms * float64(time.Millisecond)))
}
