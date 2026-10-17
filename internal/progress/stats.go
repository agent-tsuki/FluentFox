package progress

import (
	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
)

// usualTimes is the user's mean time per timed answer, per question type.
type usualTimes map[common.QuestionType]float64

func usualTimesOf(rows []learn.CardAnswerStats) usualTimes {
	ms := map[common.QuestionType]int64{}
	n := map[common.QuestionType]int{}
	for i := range rows {
		ms[rows[i].QuestionType] += rows[i].TimedMs
		n[rows[i].QuestionType] += rows[i].TimedAnswers
	}
	out := make(usualTimes, len(n))
	for qt, count := range n {
		if count > 0 {
			out[qt] = float64(ms[qt]) / float64(count)
		}
	}
	return out
}

// answerTotals sums answer stats over cards. ExpectedMs is what the timed
// answers would have taken at the user's usual time per question type, so
// question types with different speeds can be compared in one ratio.
type answerTotals struct {
	Answers      int
	Correct      int
	Reviews      int
	Passed       int
	TimedAnswers int
	TimedMs      int64
	ExpectedMs   float64
}

func (t *answerTotals) add(row *learn.CardAnswerStats, usual usualTimes) {
	t.Answers += row.Answers
	t.Correct += row.Correct
	t.Reviews += row.Reviews
	t.Passed += row.Passed
	t.TimedAnswers += row.TimedAnswers
	t.TimedMs += row.TimedMs
	t.ExpectedMs += float64(row.TimedAnswers) * usual[row.QuestionType]
}

func (t *answerTotals) merge(o *answerTotals) {
	t.Answers += o.Answers
	t.Correct += o.Correct
	t.Reviews += o.Reviews
	t.Passed += o.Passed
	t.TimedAnswers += o.TimedAnswers
	t.TimedMs += o.TimedMs
	t.ExpectedMs += o.ExpectedMs
}

// retention is the share of reviews recalled; nil without reviews.
func (t *answerTotals) retention() *float64 {
	return ratio(t.Passed, t.Reviews)
}

func (t *answerTotals) accuracy() *float64 {
	return ratio(t.Correct, t.Answers)
}

func (t *answerTotals) lapseRate() float64 {
	if t.Reviews == 0 {
		return 0
	}
	return float64(t.Reviews-t.Passed) / float64(t.Reviews)
}

// timeRatio compares the time taken with the user's usual time; 0 means no
// timed answers.
func (t *answerTotals) timeRatio() float64 {
	if t.ExpectedMs <= 0 {
		return 0
	}
	return float64(t.TimedMs) / t.ExpectedMs
}

func (t *answerTotals) avgMs() *int {
	if t.TimedAnswers == 0 {
		return nil
	}
	avg := int(t.TimedMs / int64(t.TimedAnswers))
	return &avg
}

func (t *answerTotals) usualMs() int {
	if t.TimedAnswers == 0 {
		return 0
	}
	return int(t.ExpectedMs / float64(t.TimedAnswers))
}

func ratio(part, whole int) *float64 {
	if whole == 0 {
		return nil
	}
	r := float64(part) / float64(whole)
	return &r
}

func totalsByCard(rows []learn.CardAnswerStats, usual usualTimes) map[uuid.UUID]*answerTotals {
	out := map[uuid.UUID]*answerTotals{}
	for i := range rows {
		t := out[rows[i].CardID]
		if t == nil {
			t = &answerTotals{}
			out[rows[i].CardID] = t
		}
		t.add(&rows[i], usual)
	}
	return out
}

// totalsByTrack sums the answers of the indexed cards per track.
func totalsByTrack(x *progressIndex, rows []learn.CardAnswerStats, usual usualTimes) map[common.LearningTrack]*answerTotals {
	out := map[common.LearningTrack]*answerTotals{}
	for i := range rows {
		ref, ok := x.byCard[rows[i].CardID]
		if !ok {
			continue
		}
		t := out[ref.Pool.Track]
		if t == nil {
			t = &answerTotals{}
			out[ref.Pool.Track] = t
		}
		t.add(&rows[i], usual)
	}
	return out
}
