package learn

import (
	"slices"

	"github.com/fluentfox/api/internal/common"
)

// quizAsk is one question of a kana unit quiz: which kana, which way round.
type quizAsk struct {
	item *Item
	qt   common.QuestionType
}

// isKanaUnitQuiz reports whether a test is a kana unit quiz, which asks each
// kana several times (kanaQuizPlan) and so scores every answer.
func isKanaUnitQuiz(kind common.TestKind, items []*Item) bool {
	return kind == common.TestKindUnitQuiz && len(items) > 0 && items[0].Type == common.SRSItemKana
}

// kanaUnitQuizTotal is the number of questions a kana unit quiz over n kana asks.
func kanaUnitQuizTotal(n int) int {
	return min(n*common.KANA_UNIT_QUIZ_ASKS_PER_KANA, common.KANA_UNIT_QUIZ_MAX_QUESTIONS)
}

// kanaQuizPlan lists a kana unit quiz in random order. It goes round the
// kana (shuffled) until the quiz is full, so a unit with more kana than fit
// asks as many different ones as it can. Each kana alternates direction from
// one round to the next and neighbouring kana start on opposite directions,
// so a five-kana row asks each kana twice each way. The same kana is never
// asked twice in a row.
func kanaQuizPlan(items []*Item, rnd Rand) []quizAsk {
	order := slices.Clone(items)
	rnd.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	directions := common.KANA_UNIT_QUIZ_DIRECTIONS
	total := kanaUnitQuizTotal(len(order))
	plan := make([]quizAsk, 0, total)
	for round := 0; len(plan) < total; round++ {
		for i, item := range order {
			if len(plan) == total {
				break
			}
			plan = append(plan, quizAsk{item: item, qt: directions[(i+round)%len(directions)]})
		}
	}
	rnd.Shuffle(len(plan), func(i, j int) { plan[i], plan[j] = plan[j], plan[i] })
	spreadRepeats(plan)
	return plan
}

// spreadRepeats moves questions so that no kana comes twice in a row: a
// repeat swaps with the next question about another kana, or, when only
// that kana is left, with an earlier question it can sit between.
func spreadRepeats(plan []quizAsk) {
	for i := 1; i < len(plan); i++ {
		repeat := plan[i].item
		if repeat != plan[i-1].item {
			continue
		}
		if j := slices.IndexFunc(plan[i+1:], func(a quizAsk) bool { return a.item != repeat }); j >= 0 {
			plan[i], plan[i+1+j] = plan[i+1+j], plan[i]
			continue
		}
		for k := 0; k < i-1; k++ {
			fits := plan[k].item != repeat && plan[k+1].item != repeat && (k == 0 || plan[k-1].item != repeat)
			if fits {
				plan[i], plan[k] = plan[k], plan[i]
				break
			}
		}
	}
}

// kanaQuizQuestions builds a kana unit quiz's questions.
func (q *itemQuestions) kanaQuizQuestions(items []*Item, rnd Rand) []Question {
	plan := kanaQuizPlan(items, rnd)
	out := make([]Question, 0, len(plan))
	for _, ask := range plan {
		out = append(out, q.buildAs(ask.item, common.CardFacetRecognition, ask.qt))
	}
	return out
}
