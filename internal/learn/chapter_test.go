package learn

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/content"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

var (
	chapterOne      = uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	wordStation     = uuid.MustParse("c0000000-0000-0000-0000-000000000011")
	wordTrain       = uuid.MustParse("c0000000-0000-0000-0000-000000000012")
	wordSchool      = uuid.MustParse("c0000000-0000-0000-0000-000000000013")
	wordBank        = uuid.MustParse("c0000000-0000-0000-0000-000000000014")
	wordMe          = uuid.MustParse("c0000000-0000-0000-0000-000000000015")
	sentenceTopic   = uuid.MustParse("c0000000-0000-0000-0000-000000000021")
	sentenceByTrain = uuid.MustParse("c0000000-0000-0000-0000-000000000022")
)

func (f *fakeContent) ListGrammarChapters(_ context.Context, level common.JLPTLevel) ([]content.GrammarChapter, error) {
	if level != common.JLPTLevelN5 {
		return nil, nil
	}
	return []content.GrammarChapter{{ID: chapterOne, Level: level, Slug: "ch1", Number: 1, Title: "Identity", Position: 1}}, nil
}

func (f *fakeContent) ListChapterWords(_ context.Context, level common.JLPTLevel) ([]content.ChapterWord, error) {
	if level != common.JLPTLevelN5 {
		return nil, nil
	}
	word := func(id uuid.UUID, w, kana, romaji, meaning string) content.ChapterWord {
		return content.ChapterWord{ID: id, ChapterID: chapterOne, Word: w, Kana: kana, Romaji: romaji, Meaning: meaning}
	}
	return []content.ChapterWord{
		word(wordStation, "駅", "えき", "eki", "station"),
		word(wordTrain, "電車", "でんしゃ", "densha", "train"),
		word(wordSchool, "学校", "がっこう", "gakkou", "school"),
		word(wordBank, "銀行", "ぎんこう", "ginkou", "bank"),
		word(wordMe, "私", "わたし", "watashi", "I, me"),
	}, nil
}

func (f *fakeContent) ListGrammarSentences(_ context.Context, level common.JLPTLevel) ([]content.GrammarSentence, error) {
	if level != common.JLPTLevelN5 {
		return nil, nil
	}
	return []content.GrammarSentence{
		{
			ID: sentenceTopic, ChapterID: chapterOne, Sentence: "わたし{}学生です。", Answer: "は",
			Distractors: []string{"を", "へ", "で"}, Romaji: []string{"wa"},
			Translation: "I am a student.", Explanation: "は marks the topic.",
		},
		{
			ID: sentenceByTrain, ChapterID: chapterOne, Sentence: "電車{}行きます。", Answer: "で",
			Distractors: []string{"を", "が", "の"}, AlsoAccept: []string{"でも"},
			Translation: "I go by train.", Explanation: "で marks the means.",
		},
	}, nil
}

func TestCatalogLoadsChapterTracks(t *testing.T) {
	tests := []struct {
		name     string
		track    common.LearningTrack
		items    int
		unitType common.UnitType
	}{
		{"vocabulary track holds the chapter's words", common.LearningTrackVocabN5, 5, common.UnitTypeVocabChapter},
		{"grammar track holds the chapter's sentences", common.LearningTrackGrammarN5, 2, common.UnitTypeGrammarChapter},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool := loadTestPool(tc.track)
			if len(pool.Items) != tc.items {
				t.Fatalf("items = %d, want %d", len(pool.Items), tc.items)
			}
			if len(pool.Units) != 1 || pool.Units[0].Type != tc.unitType || pool.Units[0].Slug != "ch1" {
				t.Fatalf("units = %+v, want one %s unit ch1", pool.Units, tc.unitType)
			}
		})
	}
}

func TestWordMeanings(t *testing.T) {
	tests := []struct {
		name    string
		meaning string
		want    []string
	}{
		{"single meaning stays as is", "station", []string{"station"}},
		{"comma list also accepts each part", "I, me", []string{"I, me", "I", "me"}},
		{"slash list also accepts each part", "Restroom / Toilet", []string{"Restroom / Toilet", "Restroom", "Toilet"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := wordMeanings(tc.meaning); !slices.Equal(got, tc.want) {
				t.Fatalf("wordMeanings(%q) = %v, want %v", tc.meaning, got, tc.want)
			}
		})
	}
}

func TestMatchesGap(t *testing.T) {
	topic := &SentenceItem{Answer: "は", Romaji: []string{"wa"}}
	means := &SentenceItem{Answer: "で", AlsoAccept: []string{"でも"}}
	tests := []struct {
		name     string
		sentence *SentenceItem
		given    string
		want     bool
	}{
		{"the answer in kana", topic, "は", true},
		{"a particle typed by its sound", topic, "わ", true},
		{"a listed romaji spelling", topic, " WA ", true},
		{"another accepted answer", means, "でも", true},
		{"punctuation is ignored", means, "で。", true},
		{"a wrong particle", topic, "が", false},
		{"nothing typed", topic, "  ", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sentence.matchesGap(tc.given); got != tc.want {
				t.Fatalf("matchesGap(%q) = %v, want %v", tc.given, got, tc.want)
			}
		})
	}
}

func TestChapterQuestions(t *testing.T) {
	vocab := loadTestPool(common.LearningTrackVocabN5)
	grammar := loadTestPool(common.LearningTrackGrammarN5)
	station, _ := vocab.Item(wordStation)
	topic, _ := grammar.Item(sentenceTopic)
	tests := []struct {
		name            string
		pool            *Pool
		item            *Item
		facet           common.CardFacet
		mode            common.AnswerMode
		wantType        common.QuestionType
		wantOptions     []string
		wantTranslation string
	}{
		{"a new word is a meaning choice", vocab, station, common.CardFacetMeaning, common.AnswerModeMixed,
			common.QuestionVocabMeaningMCQ, nil, ""},
		{"a sentence gap offers its own options", grammar, topic, common.CardFacetRecognition, common.AnswerModeMixed,
			common.QuestionGrammarGapMCQ, []string{"は", "を", "へ", "で"}, "I am a student."},
		{"a sentence's meaning card asks for the English", grammar, topic, common.CardFacetMeaning, common.AnswerModeMixed,
			common.QuestionGrammarTranslateMCQ, nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := NewQuestionBuilder(&fixedRand{})
			q := b.Build(&QuestionInput{
				Card: QuestionCard{ID: uuid.New(), State: srs.StateNew, Facet: tc.facet}, Item: tc.item, Pool: tc.pool, Mode: tc.mode,
			})
			if q.QuestionType != string(tc.wantType) {
				t.Fatalf("type = %s, want %s", q.QuestionType, tc.wantType)
			}
			if !slices.Contains(q.Options, q.CorrectAnswer) {
				t.Fatalf("options %v miss the answer %q", q.Options, q.CorrectAnswer)
			}
			if tc.wantOptions != nil && !sameSet(q.Options, tc.wantOptions) {
				t.Fatalf("options = %v, want %v", q.Options, tc.wantOptions)
			}
			if tc.item.Sentence != nil && (q.Sentence == nil || q.Sentence.Translation != tc.wantTranslation) {
				t.Fatalf("sentence = %+v, want translation %q", q.Sentence, tc.wantTranslation)
			}
		})
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, v := range a {
		if !slices.Contains(b, v) {
			return false
		}
	}
	return true
}

func TestCheckChapterAnswers(t *testing.T) {
	grammar := loadTestPool(common.LearningTrackGrammarN5)
	topic, _ := grammar.Item(sentenceTopic)
	text := func(s string) *AnswerRequestItem { return &AnswerRequestItem{AnswerText: &s} }
	chose := func(s string) *AnswerRequestItem { return &AnswerRequestItem{ChosenOption: &s} }
	tests := []struct {
		name string
		qt   common.QuestionType
		ans  *AnswerRequestItem
		want bool
	}{
		{"typed gap by sound", common.QuestionGrammarGapTyped, text("わ"), true},
		{"chosen gap", common.QuestionGrammarGapMCQ, chose("は"), true},
		{"wrong chosen gap", common.QuestionGrammarGapMCQ, chose("を"), false},
		{"chosen translation", common.QuestionGrammarTranslateMCQ, chose("I am a student."), true},
		{"another sentence's translation", common.QuestionGrammarTranslateMCQ, chose("I go by train."), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkAnswer(tc.qt, topic, grammar, tc.ans); got.Correct != tc.want {
				t.Fatalf("correct = %v, want %v", got.Correct, tc.want)
			}
		})
	}
}

func TestSessionSchedules(t *testing.T) {
	track := func(t common.LearningTrack) *common.LearningTrack { return &t }
	tests := []struct {
		name    string
		session StudySession
		want    bool
	}{
		{"a chapter vocab quiz schedules", StudySession{Kind: common.SessionKindUnitQuiz, Track: track(common.LearningTrackVocabN5)}, true},
		{"a chapter exercise schedules", StudySession{Kind: common.SessionKindUnitQuiz, Track: track(common.LearningTrackGrammarN5)}, true},
		{"a kana unit quiz does not", StudySession{Kind: common.SessionKindUnitQuiz, Track: track(common.LearningTrackHiragana)}, false},
		{"reviews always do", StudySession{Kind: common.SessionKindReview}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sessionSchedules(&tc.session); got != tc.want {
				t.Fatalf("sessionSchedules = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestChapterTestRules(t *testing.T) {
	tests := []struct {
		name         string
		track        common.LearningTrack
		completion   common.ChapterCompletion
		wantMark     float64
		wantComplete bool
	}{
		{"an exercise passes at the learner's mark and completes in auto mode",
			common.LearningTrackGrammarN5, common.ChapterCompletionAuto, 0.8, true},
		{"an exercise leaves completion to the learner in manual mode",
			common.LearningTrackGrammarN5, common.ChapterCompletionManual, 0.8, false},
		{"a vocab quiz pass is always recorded",
			common.LearningTrackVocabN5, common.ChapterCompletionManual, 0.8, true},
		{"kana keeps its fixed mark",
			common.LearningTrackHiragana, common.ChapterCompletionManual, common.KANA_UNIT_QUIZ_PASS_MARK, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.settings.ChapterPassPct = 80
			repo.settings.ChapterCompletion = tc.completion
			svc := NewTestScoreService(repo, NewCatalog(&fakeContent{}), nil, zap.NewNop())
			track := tc.track
			rules, err := svc.rules(context.Background(), &StudySession{UserID: testUserID, Track: &track}, common.TestKindUnitQuiz)
			if err != nil {
				t.Fatal(err)
			}
			if rules.passMark != tc.wantMark || rules.completeOnPass != tc.wantComplete {
				t.Fatalf("rules = %+v, want mark %v complete %v", rules, tc.wantMark, tc.wantComplete)
			}
		})
	}
}

func (f *fakeRepo) SetUnitCompleted(_ context.Context, _ *gorm.DB, p *UnitProgress) error {
	copied := *p
	f.unitProgress[p.UnitID] = &copied
	return f.err
}

func TestChapterCompletion(t *testing.T) {
	tests := []struct {
		name          string
		chapter       string
		completed     bool
		wantErr       error
		wantCompleted bool
		wantCards     int
	}{
		// 5 words (one meaning card each) and 2 sentences (gap and meaning cards).
		{"marking complete puts the chapter into reviews", "ch1", true, nil, true, 9},
		{"taking the mark away adds no cards", "ch1", false, nil, false, 0},
		{"an unknown chapter is refused", "ch99", true, exceptions.ErrGrammarChapterNotFound, false, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			svc := NewChapterCompletionService(repo, NewCatalog(&fakeContent{}), zap.NewNop())
			in := &ChapterCompletionInput{Level: "n5", Chapter: tc.chapter}
			in.Body.Completed = tc.completed
			got, err := svc.Set(context.Background(), testUserID, in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if (got.CompletedAt != nil) != tc.wantCompleted || got.UnitType != string(common.UnitTypeGrammarChapter) {
				t.Fatalf("progress = %+v, want completed %v on the grammar chapter", got, tc.wantCompleted)
			}
			if len(repo.cards) != tc.wantCards {
				t.Fatalf("cards = %d, want %d", len(repo.cards), tc.wantCards)
			}
			for _, c := range repo.cards {
				if c.State == srs.StateNew || c.LastReviewAt == nil {
					t.Fatalf("card %s state %d: want a learned, scheduled card", c.ID, c.State)
				}
			}
		})
	}
}
