package learn

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
)

func exerciseRow(t *testing.T, chapter uuid.UUID, kind content.ChapterExerciseKind, data any) *content.ChapterExercise {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return &content.ChapterExercise{ID: uuid.New(), ChapterID: chapter, Kind: kind, Data: raw}
}

func mustExercise(t *testing.T, row *content.ChapterExercise) Item {
	t.Helper()
	item, err := exerciseItem(row)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestExerciseMarking(t *testing.T) {
	chapter := uuid.New()
	falseItem := mustExercise(t, exerciseRow(t, chapter, content.ChapterExerciseTrueFalse, content.TrueFalseItem{
		Lines: []string{"A", "B"}, Statement: "He is a student.", Answer: false, Explanation: "why",
	}))
	transform := mustExercise(t, exerciseRow(t, chapter, content.ChapterExerciseTransform, content.TransformItem{
		Source: "学生です。", Task: "Make it negative.", Sentence: "学生{}。", Answer: "じゃありません", AlsoAccept: []string{"ではありません"}, Explanation: "why",
	}))
	tests := []struct {
		name  string
		qt    common.QuestionType
		item  *Item
		given string
		want  bool
	}{
		{"true/false right", common.QuestionExerciseTrueFalse, &falseItem, "False", true},
		{"true/false wrong", common.QuestionExerciseTrueFalse, &falseItem, "True", false},
		{"transform answer", common.QuestionExerciseTransform, &transform, "じゃありません", true},
		{"transform also accepted", common.QuestionExerciseTransform, &transform, "ではありません", true},
		{"transform wrong", common.QuestionExerciseTransform, &transform, "です", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			given := tt.given
			got := checkAnswer(tt.qt, tt.item, &Pool{}, &AnswerRequestItem{ChosenOption: &given})
			if got.Correct != tt.want || got.Chosen != nil {
				t.Fatalf("checkAnswer = %+v, want correct %v and no chosen item", got, tt.want)
			}
		})
	}
}

func wordRows(t *testing.T, chapter uuid.UUID, n int) []Item {
	t.Helper()
	items := make([]Item, 0, n)
	for i := range n {
		items = append(items, mustExercise(t, exerciseRow(t, chapter, content.ChapterExerciseWord, content.WordSentence{
			Sentence: fmt.Sprintf("文%d{}。", i), Answer: fmt.Sprintf("語%d", i), Translation: "t",
		})))
	}
	return items
}

func TestAddWordChoicesUsesOtherAnswersFromTheChapter(t *testing.T) {
	items := wordRows(t, uuid.New(), 5)
	addWordChoices(items)
	for i := range items {
		s := items[i].Sentence
		if len(s.Distractors) != common.MCQ_DISTRACTORS || slices.Contains(s.Distractors, s.Answer) {
			t.Fatalf("item %d distractors = %v (answer %s)", i, s.Distractors, s.Answer)
		}
	}
}

func TestChapterSessionAsksEveryKindAndStopsAtTheMaximum(t *testing.T) {
	chapter := uuid.New()
	sentence := Item{ID: uuid.New(), Type: common.SRSItemGrammarSentence, UnitID: chapter, Char: "A{}", Sentence: &SentenceItem{Answer: "は", Distractors: []string{"を"}, Translation: "t"}}
	sentences := []*Item{&sentence}
	var rows []*content.ChapterExercise
	add := func(n int, kind content.ChapterExerciseKind, data any) {
		for range n {
			rows = append(rows, exerciseRow(t, chapter, kind, data))
		}
	}
	wrong := []string{"x", "y", "z"}
	add(3, content.ChapterExerciseTrueFalse, content.TrueFalseItem{Statement: "S", Answer: true, Lines: []string{"A", "B"}})
	add(3, content.ChapterExerciseTransform, content.TransformItem{Sentence: "{}", Answer: "a", Source: "s", Task: "t"})
	add(3, content.ChapterExercisePicture, content.PictureItem{Sentence: "{}", Answer: "a", Options: []string{"a", "b", "c", "d"}})
	add(4, content.ChapterExerciseOrder, content.OrderItem{Words: []string{"a", "b", "c"}, Translation: "t"})
	add(4, content.ChapterExerciseListen, content.ListenItem{Text: "t", Translation: "m", Wrong: wrong, Voice: "f1"})
	add(2, content.ChapterExerciseReply, content.ReplyItem{Prompt: "p", Answer: "a", Wrong: wrong})
	add(2, content.ChapterExerciseReading, content.ReadingItem{Lines: []string{"l"}, Question: "q", Answer: "a", Wrong: wrong})
	exercises := wordRows(t, chapter, 10)
	for _, row := range rows {
		exercises = append(exercises, mustExercise(t, row))
	}
	addWordChoices(exercises)
	pool := newPool(common.LearningTrackGrammarN5, []Unit{{ID: chapter, Slug: "ch1"}}, nil)
	pool.exercises = exercises
	q := &itemQuestions{learner: &learner{}, pool: pool, builder: NewQuestionBuilder(nil)}

	got := q.chapterSession(sentences, "ch1", globalRand{})
	if len(got) != common.CHAPTER_EXERCISE_MAX_QUESTIONS {
		t.Fatalf("got %d questions, want %d", len(got), common.CHAPTER_EXERCISE_MAX_QUESTIONS)
	}
	counts := map[string]int{}
	for i := range got {
		counts[got[i].QuestionType]++
	}
	if len(counts) != 10 {
		t.Fatalf("got %d kinds of question, want 10: %v", len(counts), counts)
	}
	for kind, n := range counts {
		if n != common.CHAPTER_EXERCISE_PER_TYPE {
			t.Fatalf("%s asked %d times, want %d", kind, n, common.CHAPTER_EXERCISE_PER_TYPE)
		}
	}
	if q.chapterSession(sentences, "missing", globalRand{}) != nil {
		t.Fatal("an unknown chapter should fall back")
	}
}

func TestOrderExerciseIsMarkedOnTheJoinedWords(t *testing.T) {
	item := mustExercise(t, exerciseRow(t, uuid.New(), content.ChapterExerciseOrder, content.OrderItem{Words: []string{"これ", "は", "本", "です"}, Translation: "This is a book."}))
	right, wrong := "これは本です", "本はこれです"
	if !checkAnswer(common.QuestionExerciseOrder, &item, &Pool{}, &AnswerRequestItem{AnswerText: &right}).Correct {
		t.Fatal("the words in order should be right")
	}
	if checkAnswer(common.QuestionExerciseOrder, &item, &Pool{}, &AnswerRequestItem{AnswerText: &wrong}).Correct {
		t.Fatal("the words out of order should be wrong")
	}
}

func TestPoolsFindReachesExercisesOnly(t *testing.T) {
	item := wordRows(t, uuid.New(), 1)[0]
	pool := newPool(common.LearningTrackGrammarN5, nil, nil)
	pool.exercises = []Item{item}
	if _, found, ok := (Pools{pool}).Find(common.SRSItemGrammarExercise, item.ID); !ok || found.ID != item.ID {
		t.Fatal("exercise not found")
	}
	if _, _, ok := (Pools{pool}).Find(common.SRSItemGrammarSentence, item.ID); ok {
		t.Fatal("an exercise must not be found as a sentence")
	}
	if len(pool.chapterItems(item.UnitID)) != 0 {
		t.Fatal("exercises must stay out of the chapter's learnable items")
	}
}
