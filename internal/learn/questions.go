package learn

import (
	"math/rand/v2"
	"slices"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

// Rand is satisfied by *rand.Rand; tests pass a fixed sequence.
type Rand interface {
	IntN(n int) int
	Shuffle(n int, swap func(i, j int))
}

type globalRand struct{}

// Question order and options need no cryptographic randomness.
func (globalRand) IntN(n int) int { return rand.IntN(n) } //nolint:gosec // see above

func (globalRand) Shuffle(n int, swap func(i, j int)) { rand.Shuffle(n, swap) } //nolint:gosec // see above

// Int64N lets globalRand spread the due dates of known cards (srs.RandSource).
func (globalRand) Int64N(n int64) int64 { return rand.Int64N(n) } //nolint:gosec // see above

type QuestionCard struct {
	ID        uuid.UUID
	State     srs.State
	Stability float64
	Facet     common.CardFacet
	IsLeech   bool
}

type QuestionInput struct {
	Card QuestionCard
	Item *Item
	Pool *Pool
	// Items the user picked by mistake for this item, most often first.
	Confused []uuid.UUID
	Mode     common.AnswerMode
}

// QuestionBuilder picks the question type for a card and its wrong options
// (docs/learning-plan.md section 7.3).
type QuestionBuilder struct {
	rnd Rand
}

func NewQuestionBuilder(rnd Rand) *QuestionBuilder {
	if rnd == nil {
		rnd = globalRand{}
	}
	return &QuestionBuilder{rnd: rnd}
}

func (b *QuestionBuilder) Build(in *QuestionInput) Question {
	return b.BuildAs(in, b.chooseType(in))
}

// BuildAs builds a question of a type the caller chose, such as a speed round
// or a look-alike drill. qt must fit the card's facet.
func (b *QuestionBuilder) BuildAs(in *QuestionInput, qt common.QuestionType) Question {
	answer := questionAnswerKinds[qt]
	q := Question{
		CardID:          in.Card.ID,
		ItemType:        string(in.Item.Type),
		ItemID:          in.Item.ID,
		Facet:           string(in.Card.Facet),
		QuestionType:    string(qt),
		Prompt:          itemValue(in.Item, questionPromptKinds[qt]),
		CorrectAnswer:   itemValue(in.Item, answer),
		AcceptedAnswers: acceptedValues(in.Item, answer),
		CardState:       int(in.Card.State),
		Char:            in.Item.Char,
		IsLeech:         in.Card.IsLeech,
	}
	if !common.QUESTION_TYPE_RULES[qt].Typed {
		q.Options = b.options(in, answer, q.CorrectAnswer)
	}
	return q
}

// New cards always get plain multiple choice; the variety types come in once
// the card has been seen.
func (b *QuestionBuilder) chooseType(in *QuestionInput) common.QuestionType {
	if in.Card.State != srs.StateNew && b.rnd.IntN(common.VARIETY_QUESTION_ONE_IN) == 0 {
		if qt, ok := b.varietyType(in); ok {
			return qt
		}
	}
	typed := allowsTyped(&in.Card, in.Mode)
	switch in.Card.Facet {
	case common.CardFacetMeaning:
		if typed {
			return common.QuestionKanjiMeaningTyped
		}
		return common.QuestionKanjiMeaningMCQ
	case common.CardFacetReading:
		if typed {
			return common.QuestionKanjiReadingTyped
		}
		return common.QuestionKanjiReadingMCQ
	default:
		if typed {
			return common.QuestionKanaTyped
		}
		if b.rnd.IntN(2) == 0 {
			return common.QuestionKanaMCQRomaji
		}
		return common.QuestionKanaMCQKana
	}
}

func (b *QuestionBuilder) varietyType(in *QuestionInput) (common.QuestionType, bool) {
	hasLookAlikes := len(in.Item.LookAlikes) > 0
	switch in.Card.Facet {
	case common.CardFacetMeaning:
		return common.QuestionKanjiLookalike, hasLookAlikes
	case common.CardFacetReading:
		return common.QuestionKanjiListen, true
	default:
		if hasLookAlikes && b.rnd.IntN(2) == 0 {
			return common.QuestionKanaLookalike, true
		}
		return common.QuestionKanaListen, true
	}
}

func allowsTyped(card *QuestionCard, mode common.AnswerMode) bool {
	switch mode {
	case common.AnswerModeChoice:
		return false
	case common.AnswerModeTyped:
		return card.State != srs.StateNew
	default:
		return card.State == srs.StateReview && card.Stability >= common.TYPED_MIN_STABILITY_DAYS
	}
}

func (b *QuestionBuilder) options(in *QuestionInput, kind answerKind, correct string) []string {
	options := []string{correct}
	for _, d := range b.distractors(in, kind, correct) {
		options = append(options, itemValue(d, kind))
	}
	b.rnd.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	return options
}

// distractors takes wrong options in order of usefulness: the user's own
// past mix-ups, then look-alikes, then the same kana row or kanji unit, then
// anything else. Items that would also be right are never used.
func (b *QuestionBuilder) distractors(in *QuestionInput, kind answerKind, correct string) []*Item {
	picked := make([]*Item, 0, common.MCQ_DISTRACTORS)
	used := map[string]bool{correct: true}
	for _, other := range b.candidates(in) {
		value := itemValue(other, kind)
		if other.ID == in.Item.ID || value == "" || used[value] || sameAnswer(in.Card.Facet, in.Item, other) {
			continue
		}
		used[value] = true
		picked = append(picked, other)
		if len(picked) == common.MCQ_DISTRACTORS {
			break
		}
	}
	return picked
}

func (b *QuestionBuilder) candidates(in *QuestionInput) []*Item {
	items := in.Pool.Items
	out := make([]*Item, 0, len(in.Confused)+len(in.Item.LookAlikes)+2*len(items))
	for _, id := range slices.Concat(in.Confused, in.Item.LookAlikes) {
		if other, ok := in.Pool.Item(id); ok {
			out = append(out, other)
		}
	}
	for i := range items {
		if items[i].UnitID == in.Item.UnitID {
			out = append(out, &items[i])
		}
	}
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	b.rnd.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	for _, i := range order {
		out = append(out, &items[i])
	}
	return out
}
