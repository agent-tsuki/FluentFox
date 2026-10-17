package learn

import (
	"slices"
	"strings"

	"github.com/fluentfox/api/internal/common"
)

// answerCheck is the server's own marking of an answer. Chosen is the item
// whose value the user picked in a wrong multiple-choice answer.
type answerCheck struct {
	Correct      bool
	TypoDistance int
	Chosen       *Item
}

func checkAnswer(qt common.QuestionType, item *Item, pool *Pool, ans *AnswerRequestItem) answerCheck {
	kind := questionAnswerKinds[qt]
	given := givenAnswer(ans)
	switch {
	case kind == kindNone:
		return answerCheck{Correct: !ans.AnswerRevealed}
	case kind == kindMeaning && common.QUESTION_TYPE_RULES[qt].Typed:
		m := MatchMeaning(normalizeMeaningAnswer(given), meaningSet(item), otherMeanings(pool, item))
		return answerCheck{Correct: m.Correct, TypoDistance: m.TypoDistance}
	}
	if matchesItem(kind, item, given) {
		return answerCheck{Correct: true}
	}
	if ans.ChosenOption == nil {
		return answerCheck{}
	}
	return answerCheck{Chosen: findChosen(pool, kind, item, given)}
}

func givenAnswer(ans *AnswerRequestItem) string {
	if ans.ChosenOption != nil && *ans.ChosenOption != "" {
		return *ans.ChosenOption
	}
	if ans.AnswerText != nil {
		return *ans.AnswerText
	}
	return ""
}

func matchesItem(kind answerKind, item *Item, given string) bool {
	if strings.TrimSpace(given) == "" {
		return false
	}
	switch kind {
	case kindRomaji:
		return slices.Contains(romajiSet(item), normalizeRomaji(given))
	case kindChar:
		return strings.TrimSpace(given) == item.Char
	case kindMeaning:
		return slices.Contains(meaningSet(item), normalizeMeaningAnswer(given))
	case kindReading:
		return slices.Contains(readingSet(item), normalizeReading(given))
	default:
		return false
	}
}

// Options are built from itemValue, so the picked text names one item.
func findChosen(pool *Pool, kind answerKind, target *Item, given string) *Item {
	for i := range pool.Items {
		other := &pool.Items[i]
		if other.ID != target.ID && itemValue(other, kind) == given {
			return other
		}
	}
	return nil
}

func otherMeanings(pool *Pool, target *Item) []string {
	var out []string
	for i := range pool.Items {
		if pool.Items[i].ID != target.ID {
			out = append(out, meaningSet(&pool.Items[i])...)
		}
	}
	return out
}
