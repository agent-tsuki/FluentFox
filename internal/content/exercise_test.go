package content

import (
	"encoding/json"
	"errors"
	"testing"
)

const minWordSentencesPerChapter = 10

// The shipped file must pass the sync's checks and cover real chapters, so a
// bad edit fails here and not on deploy.
func TestExerciseFilesAreValid(t *testing.T) {
	var file ExerciseFile
	readEmbedded(t, "exercises/n5.json", &file)
	var chapters ChapterFile
	readEmbedded(t, "chapters/n5.json", &chapters)
	slugs := map[string]bool{}
	for _, c := range chapters.Chapters {
		slugs[c.Slug] = true
	}
	if err := CheckExerciseFile(&file); err != nil {
		t.Fatal(err)
	}
	if len(file.Chapters) != len(chapters.Chapters) {
		t.Fatalf("%d chapters have exercises, want all %d", len(file.Chapters), len(chapters.Chapters))
	}
	words := map[string]map[string]bool{}
	for _, c := range chapters.Chapters {
		words[c.Slug] = map[string]bool{}
		for _, w := range c.Words {
			words[c.Slug][w.Word] = true
		}
	}
	for i := range file.Chapters {
		c := &file.Chapters[i]
		if !slugs[c.Slug] {
			t.Errorf("no chapter %q", c.Slug)
		}
		if len(c.Words) < minWordSentencesPerChapter {
			t.Errorf("%s has %d word sentences, want at least %d", c.Slug, len(c.Words), minWordSentencesPerChapter)
		}
		seen := map[string]bool{}
		for _, w := range c.Words {
			if !words[c.Slug][w.Answer] {
				t.Errorf("%s: answer %q is not one of the chapter's words", c.Slug, w.Answer)
			}
			if seen[w.Answer] {
				t.Errorf("%s: %q is the answer twice, so the word bank would be ambiguous", c.Slug, w.Answer)
			}
			seen[w.Answer] = true
		}
	}
}

func TestCheckExerciseChapter(t *testing.T) {
	goodPicture := PictureItem{Image: "/exercises/a.jpg", Alt: "a", Sentence: "{}です", Answer: "ア", Options: []string{"ア", "イ", "ウ", "エ"}, Translation: "t"}
	goodTransform := TransformItem{Source: "s", Task: "t", Sentence: "a{}", Answer: "b", Explanation: "e"}
	picture := func(change func(*PictureItem)) PictureItem {
		p := goodPicture
		change(&p)
		return p
	}
	tests := []struct {
		name string
		c    ExerciseFileChapter
		want error
	}{
		{"valid items", ExerciseFileChapter{
			TrueFalse:  []TrueFalseItem{{Lines: []string{"A", "B"}, Statement: "s", Explanation: "e"}},
			Transforms: []TransformItem{goodTransform},
			Pictures:   []PictureItem{goodPicture},
		}, nil},
		{"true/false with one line", ExerciseFileChapter{TrueFalse: []TrueFalseItem{{Lines: []string{"A"}, Statement: "s", Explanation: "e"}}}, errTrueFalse},
		{"word sentence without a gap", ExerciseFileChapter{Words: []WordSentence{{Sentence: "a", Answer: "b", Translation: "t"}}}, errWordSentence},
		{"transform without a gap", ExerciseFileChapter{Transforms: []TransformItem{{Source: "s", Task: "t", Sentence: "a", Answer: "b", Explanation: "e"}}}, errTransform},
		{"picture outside /exercises/", ExerciseFileChapter{Pictures: []PictureItem{picture(func(p *PictureItem) { p.Image = "https://x/a.jpg" })}}, errPicture},
		{"picture whose options lack the answer", ExerciseFileChapter{Pictures: []PictureItem{picture(func(p *PictureItem) { p.Answer = "オ" })}}, errPicture},
		{"picture with repeated options", ExerciseFileChapter{Pictures: []PictureItem{picture(func(p *PictureItem) { p.Options = []string{"ア", "ア", "イ", "ウ"} })}}, errPicture},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.c
			if err := checkExerciseChapter(&c); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestExerciseRowsRoundTrip(t *testing.T) {
	c := ExerciseFileChapter{
		TrueFalse:  []TrueFalseItem{{Lines: []string{"A", "B"}, Statement: "s", Answer: true, Explanation: "e"}},
		Transforms: []TransformItem{{Source: "s", Task: "t", Sentence: "a{}", Answer: "b", Explanation: "e"}},
	}
	rows, err := exerciseRows(&c)
	if err != nil {
		t.Fatal(err)
	}
	out := ChapterContentResponse{}
	for i := range rows {
		if !json.Valid(rows[i].Data) {
			t.Fatalf("row %d is not JSON", i)
		}
		if err := addExercise(&rows[i], &out); err != nil {
			t.Fatal(err)
		}
	}
	if len(out.TrueFalse) != 1 || !out.TrueFalse[0].Answer || len(out.Transforms) != 1 || out.Transforms[0].Answer != "b" {
		t.Fatalf("out = %+v", out)
	}
}
