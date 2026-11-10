package learn

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
)

// ExerciseItem is what a chapter exercise has besides its SentenceItem, which
// holds its answer, wrong answers and explanation so it is marked as a
// sentence's gap is.
type ExerciseItem struct {
	QuestionType common.QuestionType
	Kind         content.ChapterExerciseKind
	Lines        []string
	Source       string
	Task         string
	Image        string
	Alt          string
	Tiles        []string
	Voice        string
	Say          string
}

func (e *ExerciseItem) question() *ExerciseQuestion {
	return &ExerciseQuestion{Lines: e.Lines, Source: e.Source, Task: e.Task, Image: e.Image, Alt: e.Alt, Tiles: slices.Clone(e.Tiles), Voice: e.Voice, Say: e.Say}
}

const (
	trueFalseTrue  = "True"
	trueFalseFalse = "False"
)

func (p *Pool) exercise(id uuid.UUID) (*Item, bool) {
	for i := range p.exercises {
		if p.exercises[i].ID == id {
			return &p.exercises[i], true
		}
	}
	return nil, false
}

func (c *Catalog) exerciseItems(ctx context.Context, level common.JLPTLevel) ([]Item, error) {
	rows, err := c.content.ListLevelExercises(ctx, level)
	if err != nil {
		return nil, fmt.Errorf("chapter exercises: %w", err)
	}
	items := make([]Item, 0, len(rows))
	for i := range rows {
		item, err := exerciseItem(&rows[i])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	addWordChoices(items)
	return items, nil
}

func exerciseItem(row *content.ChapterExercise) (Item, error) {
	item := Item{Type: common.SRSItemGrammarExercise, ID: row.ID, UnitID: row.ChapterID}
	var err error
	switch row.Kind {
	case content.ChapterExerciseTrueFalse:
		var d content.TrueFalseItem
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Statement
			answer, wrong := trueFalseTrue, trueFalseFalse
			if !d.Answer {
				answer, wrong = wrong, answer
			}
			item.Sentence = &SentenceItem{Answer: answer, Distractors: []string{wrong}, Explanation: d.Explanation}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExerciseTrueFalse, Kind: row.Kind, Lines: d.Lines}
		}
	case content.ChapterExerciseTransform:
		var d content.TransformItem
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Sentence
			item.Sentence = &SentenceItem{Answer: d.Answer, AlsoAccept: d.AlsoAccept, Explanation: d.Explanation}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExerciseTransform, Kind: row.Kind, Source: d.Source, Task: d.Task}
		}
	case content.ChapterExerciseWord:
		var d content.WordSentence
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Sentence
			item.Sentence = &SentenceItem{Answer: d.Answer, Translation: d.Translation}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExerciseWordChoice, Kind: row.Kind}
		}
	case content.ChapterExercisePicture:
		var d content.PictureItem
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Sentence
			wrong := slices.DeleteFunc(slices.Clone(d.Options), func(o string) bool { return o == d.Answer })
			item.Sentence = &SentenceItem{Answer: d.Answer, Distractors: wrong, Translation: d.Translation}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExercisePicture, Kind: row.Kind, Image: d.Image, Alt: d.Alt}
		}
	case content.ChapterExerciseOrder:
		var d content.OrderItem
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Translation
			item.Sentence = &SentenceItem{Answer: strings.Join(d.Words, "")}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExerciseOrder, Kind: row.Kind, Tiles: d.Words}
		}
	case content.ChapterExerciseListen:
		var d content.ListenItem
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Text
			item.Sentence = &SentenceItem{Answer: d.Translation, Distractors: d.Wrong}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExerciseListen, Kind: row.Kind, Voice: d.Voice, Say: d.Say}
		}
	case content.ChapterExerciseReply:
		var d content.ReplyItem
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Prompt
			item.Sentence = &SentenceItem{Answer: d.Answer, Distractors: d.Wrong, Explanation: d.Explanation}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExerciseReply, Kind: row.Kind}
		}
	case content.ChapterExerciseReading:
		var d content.ReadingItem
		if err = json.Unmarshal(row.Data, &d); err == nil {
			item.Char = d.Question
			item.Sentence = &SentenceItem{Answer: d.Answer, Distractors: d.Wrong}
			item.Exercise = &ExerciseItem{QuestionType: common.QuestionExerciseReading, Kind: row.Kind, Lines: d.Lines}
		}
	default:
		err = fmt.Errorf("unknown kind %q", row.Kind)
	}
	if err != nil {
		return Item{}, fmt.Errorf("decode exercise %s: %w", row.ID, err)
	}
	return item, nil
}

// addWordChoices gives each word exercise the next answers in its chapter as
// wrong options, since the content only names the right word.
func addWordChoices(items []Item) {
	byChapter := map[uuid.UUID][]*Item{}
	for i := range items {
		if items[i].Exercise.Kind == content.ChapterExerciseWord {
			byChapter[items[i].UnitID] = append(byChapter[items[i].UnitID], &items[i])
		}
	}
	for _, words := range byChapter {
		for at, item := range words {
			for step := 1; step < len(words) && len(item.Sentence.Distractors) < common.MCQ_DISTRACTORS; step++ {
				answer := words[(at+step)%len(words)].Sentence.Answer
				if answer != item.Sentence.Answer && !slices.Contains(item.Sentence.Distractors, answer) {
					item.Sentence.Distractors = append(item.Sentence.Distractors, answer)
				}
			}
		}
	}
}

// exerciseKinds is the order a chapter's kinds of exercise are asked in
// before the session is shuffled.
var exerciseKinds = []content.ChapterExerciseKind{
	content.ChapterExerciseTrueFalse, content.ChapterExerciseTransform, content.ChapterExerciseWord, content.ChapterExercisePicture,
	content.ChapterExerciseOrder, content.ChapterExerciseListen, content.ChapterExerciseReply, content.ChapterExerciseReading,
}

// chapterSession is a grammar chapter's exercise: a few of each kind of
// question, so every kind is practised, topped up from what is left when a
// chapter is short of one kind. It returns nil for a chapter with no
// exercises, which then asks its sentences as before.
func (q *itemQuestions) chapterSession(sentences []*Item, slug string, rnd Rand) []Question {
	unit, ok := q.pool.UnitBySlug(slug)
	if !ok {
		return nil
	}
	byKind := map[content.ChapterExerciseKind][]*Item{}
	for i := range q.pool.exercises {
		if item := &q.pool.exercises[i]; item.UnitID == unit.ID {
			byKind[item.Exercise.Kind] = append(byKind[item.Exercise.Kind], item)
		}
	}
	if len(byKind) == 0 {
		return nil
	}
	out := q.sentenceQuestions(sentences, rnd)
	var spare []*Item
	for _, kind := range exerciseKinds {
		items := byKind[kind]
		rnd.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
		take := min(common.CHAPTER_EXERCISE_PER_TYPE, len(items))
		for _, item := range items[:take] {
			out = append(out, q.build(item, common.CardFacetRecognition))
		}
		spare = append(spare, items[take:]...)
	}
	rnd.Shuffle(len(spare), func(i, j int) { spare[i], spare[j] = spare[j], spare[i] })
	for _, item := range spare {
		if len(out) >= common.CHAPTER_EXERCISE_MAX_QUESTIONS {
			break
		}
		out = append(out, q.build(item, common.CardFacetRecognition))
	}
	return out[:min(common.CHAPTER_EXERCISE_MAX_QUESTIONS, len(out))]
}

// sentenceQuestions asks the gap and the meaning of different sentences where
// the chapter has enough of them.
func (q *itemQuestions) sentenceQuestions(sentences []*Item, rnd Rand) []Question {
	if len(sentences) == 0 {
		return nil
	}
	order := slices.Clone(sentences)
	rnd.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	out := make([]Question, 0, 2*common.CHAPTER_EXERCISE_PER_TYPE)
	for i := range common.CHAPTER_EXERCISE_PER_TYPE {
		out = append(out,
			q.build(order[i%len(order)], common.CardFacetRecognition),
			q.build(order[(i+common.CHAPTER_EXERCISE_PER_TYPE)%len(order)], common.CardFacetMeaning),
		)
	}
	return out
}
