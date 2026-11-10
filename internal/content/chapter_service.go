package content

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

// ChapterContentReader is what ChapterContentService needs.
type ChapterContentReader interface {
	ListGrammarChapters(ctx context.Context, level common.JLPTLevel) ([]GrammarChapter, error)
	ListChapterWords(ctx context.Context, level common.JLPTLevel) ([]ChapterWord, error)
	ListGrammarSentences(ctx context.Context, level common.JLPTLevel) ([]GrammarSentence, error)
	ListChapterPassages(ctx context.Context, chapterID uuid.UUID) ([]ChapterPassage, error)
	ListPassageLines(ctx context.Context, chapterID uuid.UUID) ([]ChapterPassageLine, error)
	ListChapterExercises(ctx context.Context, chapterID uuid.UUID) ([]ChapterExercise, error)
}

var _ ChapterContentReader = (*Repository)(nil)

type ChapterContentInput struct {
	Level   string `path:"level"   example:"n5"  doc:"Lower-case JLPT level id"`
	Chapter string `path:"chapter" example:"ch5" doc:"Chapter slug"`
}

type ChapterContentResponse struct {
	Slug      string                `json:"slug"`
	Number    int                   `json:"number"`
	Title     string                `json:"title"`
	Words     []ChapterWordDTO      `json:"words"`
	Sentences []ChapterSentenceText `json:"sentences" doc:"The practice sentences read in full"`
	Passages  []ChapterPassageDTO   `json:"passages"  doc:"The Listening tab: short dialogues and passages, in order"`
	// The Exercise tab's practice material: answers included, as nothing here is graded by the server.
	Gaps          []ChapterGapDTO `json:"gaps"        doc:"The practice sentences with their gap and choices"`
	TrueFalse     []TrueFalseItem `json:"true_false"`
	Transforms    []TransformItem `json:"transforms"`
	Pictures      []PictureItem   `json:"pictures"`
	WordSentences []WordSentence  `json:"word_sentences" doc:"Sentences to drop a chapter word into"`
}

// ChapterGapDTO is a practice sentence with {} where its answer goes.
type ChapterGapDTO struct {
	Sentence    string   `json:"sentence"`
	Answer      string   `json:"answer"`
	Distractors []string `json:"distractors"`
	Translation string   `json:"translation"`
	Explanation string   `json:"explanation"`
}

// ChapterPassageDTO is one dialogue or passage. Its text is written with
// readings: words separated by spaces and each kanji run followed by its
// reading in brackets, e.g. "私[わたし] は 学生[がくせい] です。".
type ChapterPassageDTO struct {
	Kind     ListeningPassageKind `json:"kind"     enum:"dialogue,monologue"`
	Title    string               `json:"title"`
	Speakers []string             `json:"speakers" doc:"A dialogue's 2-4 speakers (with readings); a monologue's reader, or none"`
	// SpeakerVoices pairs with Speakers: the voice each one is played by.
	SpeakerVoices []string         `json:"speaker_voices" doc:"Each speaker's voice key (e.g. f1, m2) for /audio/speech; empty in older content"`
	Lines         []ChapterLineDTO `json:"lines"`
}

type ChapterLineDTO struct {
	Speaker     *int    `json:"speaker,omitempty" doc:"In a dialogue, which speaker (index into speakers); missing in a monologue"`
	Text        string  `json:"text"`
	Translation string  `json:"translation"`
	Say         *string `json:"say,omitempty" doc:"Text to speak instead, where the voice engine misreads the line as written"`
}

type ChapterWordDTO struct {
	Word    string  `json:"word"`
	Kana    string  `json:"kana"`
	Romaji  string  `json:"romaji"`
	Meaning string  `json:"meaning"`
	Hint    *string `json:"hint,omitempty"`
	Say     *string `json:"say,omitempty" doc:"Text to speak instead of kana, where the voice engine misreads the kana alone"`
}

type ChapterSentenceText struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

// ChapterContentService serves one chapter's words and sentences to the
// Vocabulary and Listening tabs.
type ChapterContentService struct {
	repo   ChapterContentReader
	logger *zap.Logger
}

func NewChapterContentService(repo ChapterContentReader, log *zap.Logger) *ChapterContentService {
	return &ChapterContentService{repo: repo, logger: log}
}

func (s *ChapterContentService) Get(ctx context.Context, rawLevel, slug string) (ChapterContentResponse, error) {
	level := common.JLPTLevel(strings.ToUpper(rawLevel))
	chapters, err := s.repo.ListGrammarChapters(ctx, level)
	if err != nil {
		return ChapterContentResponse{}, fmt.Errorf("get chapter content: %w", err)
	}
	var chapter *GrammarChapter
	for i := range chapters {
		if chapters[i].Slug == slug {
			chapter = &chapters[i]
		}
	}
	if chapter == nil {
		return ChapterContentResponse{}, exceptions.ErrGrammarChapterNotFound
	}
	out := ChapterContentResponse{
		Slug: chapter.Slug, Number: chapter.Number, Title: chapter.Title,
		Words: []ChapterWordDTO{}, Sentences: []ChapterSentenceText{}, Passages: []ChapterPassageDTO{},
		Gaps: []ChapterGapDTO{}, TrueFalse: []TrueFalseItem{}, Transforms: []TransformItem{}, Pictures: []PictureItem{}, WordSentences: []WordSentence{},
	}
	if err := s.addWords(ctx, level, chapter, &out); err != nil {
		return ChapterContentResponse{}, err
	}
	if err := s.addSentences(ctx, level, chapter, &out); err != nil {
		return ChapterContentResponse{}, err
	}
	if err := s.addPassages(ctx, chapter, &out); err != nil {
		return ChapterContentResponse{}, err
	}
	if err := s.addExercises(ctx, chapter, &out); err != nil {
		return ChapterContentResponse{}, err
	}
	return out, nil
}

func (s *ChapterContentService) addExercises(ctx context.Context, chapter *GrammarChapter, out *ChapterContentResponse) error {
	items, err := s.repo.ListChapterExercises(ctx, chapter.ID)
	if err != nil {
		return fmt.Errorf("get chapter exercises: %w", err)
	}
	for i := range items {
		if err := addExercise(&items[i], out); err != nil {
			return err
		}
	}
	return nil
}

func addExercise(e *ChapterExercise, out *ChapterContentResponse) error {
	var err error
	switch e.Kind {
	case ChapterExerciseTrueFalse:
		var item TrueFalseItem
		if err = json.Unmarshal(e.Data, &item); err == nil {
			out.TrueFalse = append(out.TrueFalse, item)
		}
	case ChapterExerciseTransform:
		var item TransformItem
		if err = json.Unmarshal(e.Data, &item); err == nil {
			out.Transforms = append(out.Transforms, item)
		}
	case ChapterExerciseWord:
		var item WordSentence
		if err = json.Unmarshal(e.Data, &item); err == nil {
			out.WordSentences = append(out.WordSentences, item)
		}
	case ChapterExercisePicture:
		var item PictureItem
		if err = json.Unmarshal(e.Data, &item); err == nil {
			out.Pictures = append(out.Pictures, item)
		}
	}
	if err != nil {
		return fmt.Errorf("decode %s exercise %d: %w", e.Kind, e.Position, err)
	}
	return nil
}

func (s *ChapterContentService) addWords(ctx context.Context, level common.JLPTLevel, chapter *GrammarChapter, out *ChapterContentResponse) error {
	words, err := s.repo.ListChapterWords(ctx, level)
	if err != nil {
		return fmt.Errorf("get chapter words: %w", err)
	}
	for i := range words {
		w := &words[i]
		if w.ChapterID == chapter.ID {
			out.Words = append(out.Words, ChapterWordDTO{Word: w.Word, Kana: w.Kana, Romaji: w.Romaji, Meaning: w.Meaning, Hint: w.Hint, Say: w.Say})
		}
	}
	return nil
}

func (s *ChapterContentService) addSentences(ctx context.Context, level common.JLPTLevel, chapter *GrammarChapter, out *ChapterContentResponse) error {
	sentences, err := s.repo.ListGrammarSentences(ctx, level)
	if err != nil {
		return fmt.Errorf("get chapter sentences: %w", err)
	}
	for i := range sentences {
		st := &sentences[i]
		if st.ChapterID == chapter.ID {
			text := strings.Replace(st.Sentence, "{}", st.Answer, 1)
			out.Sentences = append(out.Sentences, ChapterSentenceText{Text: text, Translation: st.Translation})
			out.Gaps = append(out.Gaps, ChapterGapDTO{
				Sentence: st.Sentence, Answer: st.Answer, Distractors: st.Distractors, Translation: st.Translation, Explanation: st.Explanation,
			})
		}
	}
	return nil
}

func (s *ChapterContentService) addPassages(ctx context.Context, chapter *GrammarChapter, out *ChapterContentResponse) error {
	passages, err := s.repo.ListChapterPassages(ctx, chapter.ID)
	if err != nil {
		return fmt.Errorf("get chapter passages: %w", err)
	}
	lines, err := s.repo.ListPassageLines(ctx, chapter.ID)
	if err != nil {
		return fmt.Errorf("get chapter passage lines: %w", err)
	}
	byPassage := make(map[uuid.UUID][]ChapterLineDTO, len(passages))
	for i := range lines {
		l := &lines[i]
		byPassage[l.PassageID] = append(byPassage[l.PassageID], ChapterLineDTO{Speaker: l.Speaker, Text: l.Text, Translation: l.Translation, Say: l.Say})
	}
	for i := range passages {
		p := &passages[i]
		out.Passages = append(out.Passages, ChapterPassageDTO{
			Kind: p.Kind, Title: p.Title, Speakers: nonNil(p.Speakers), SpeakerVoices: nonNil(p.SpeakerVoices),
			Lines: nonNilLines(byPassage[p.ID]),
		})
	}
	return nil
}

func nonNilLines(lines []ChapterLineDTO) []ChapterLineDTO {
	if lines == nil {
		return []ChapterLineDTO{}
	}
	return lines
}
