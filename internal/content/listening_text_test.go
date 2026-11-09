package content

import (
	"encoding/json"
	"errors"
	"testing"

	contentfiles "github.com/fluentfox/api/content"
)

func TestCheckRubyText(t *testing.T) {
	tests := []struct {
		text string
		want error
	}{
		{"私[わたし] は 学生[がくせい] です。", nil},
		{"7時[しちじ] に 起[お]きます。", nil},
		{"引[ひ]っ越[こ]し", nil},
		{"時々[ときどき]", nil},
		{"ラーメン が 好[す]き", nil},
		{"", errRubyEmpty},
		{"私 は 学生[がくせい]", errRubyMissingRead},
		{"学生", errRubyMissingRead},
		{"は[は]", errRubyNoBase},
		{"私[watashi]", errRubyBadReading},
		{"私[]", errRubyBadReading},
		{"私[わたし", errRubyUnclosed},
		{"私]", errRubyStray},
	}
	for _, tc := range tests {
		if err := checkRubyText(tc.text); !errors.Is(err, tc.want) {
			t.Errorf("checkRubyText(%q) = %v, want %v", tc.text, err, tc.want)
		}
	}
}

func TestCheckPassage(t *testing.T) {
	zero, two, three := 0, 2, 3
	line := func(speaker *int) ListeningFileLine {
		return ListeningFileLine{Speaker: speaker, Text: "はい。", Translation: "Yes."}
	}
	withSay := func(l ListeningFileLine, say string) ListeningFileLine {
		l.Say = &say
		return l
	}
	sp := func(names ...string) []ListeningFileSpeaker {
		out := make([]ListeningFileSpeaker, len(names))
		for i, n := range names {
			out[i] = ListeningFileSpeaker{Name: n, Voice: "f1"}
		}
		return out
	}
	tests := []struct {
		name string
		p    ListeningFilePassage
		want error
	}{
		{"a dialogue", ListeningFilePassage{Kind: ListeningDialogue, Title: "t", Speakers: sp("アナ", "ケン"), Lines: []ListeningFileLine{line(&zero)}}, nil},
		{"four people", ListeningFilePassage{Kind: ListeningDialogue, Title: "t", Speakers: sp("アナ", "ケン", "リン", "先生[せんせい]"), Lines: []ListeningFileLine{line(&three)}}, nil},
		{"a monologue", ListeningFilePassage{Kind: ListeningMonologue, Title: "t", Lines: []ListeningFileLine{line(nil)}}, nil},
		{"a monologue with its reader", ListeningFilePassage{Kind: ListeningMonologue, Title: "t", Speakers: sp("リン"), Lines: []ListeningFileLine{line(nil)}}, nil},
		{"one speaker", ListeningFilePassage{Kind: ListeningDialogue, Title: "t", Speakers: sp("アナ"), Lines: []ListeningFileLine{line(&zero)}}, errPassageSpeakers},
		{"five people", ListeningFilePassage{Kind: ListeningDialogue, Title: "t", Speakers: sp("ア", "イ", "ウ", "エ", "オ"), Lines: []ListeningFileLine{line(&zero)}}, errPassageSpeakers},
		{"a speaker who is not in the scene", ListeningFilePassage{Kind: ListeningDialogue, Title: "t", Speakers: sp("アナ", "ケン"), Lines: []ListeningFileLine{line(&two)}}, errLineSpeaker},
		{"a speaker line in a monologue", ListeningFilePassage{Kind: ListeningMonologue, Title: "t", Lines: []ListeningFileLine{line(&zero)}}, errLineSpeaker},
		{"a bad voice key", ListeningFilePassage{Kind: ListeningDialogue, Title: "t", Speakers: []ListeningFileSpeaker{{Name: "アナ", Voice: "F 1"}, {Name: "ケン", Voice: "m1"}}, Lines: []ListeningFileLine{line(&zero)}}, errSpeakerVoice},
		{"no lines", ListeningFilePassage{Kind: ListeningMonologue, Title: "t"}, errPassageNoLines},
		{"an unknown kind", ListeningFilePassage{Kind: "song", Title: "t", Lines: []ListeningFileLine{line(nil)}}, errPassageKind},
		{"a line with what to say instead", ListeningFilePassage{Kind: ListeningMonologue, Title: "t", Lines: []ListeningFileLine{withSay(line(nil), "はい。")}}, nil},
		{"a say written with readings", ListeningFilePassage{Kind: ListeningMonologue, Title: "t", Lines: []ListeningFileLine{withSay(line(nil), "家[うち]。")}}, errLineSay},
		{"a say with other sentences", ListeningFilePassage{Kind: ListeningMonologue, Title: "t", Lines: []ListeningFileLine{withSay(line(nil), "はい。はい。")}}, errLineSay},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkPassage(&tc.p); !errors.Is(err, tc.want) {
				t.Fatalf("checkPassage = %v, want %v", err, tc.want)
			}
		})
	}
}

// The shipped listening files must pass the same checks the sync runs, and
// name only chapters that exist, so a bad edit fails here and not on deploy.
func TestListeningFilesAreValid(t *testing.T) {
	paths, err := contentfiles.FS.ReadDir("listening")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no listening files: %v", err)
	}
	var chapters ChapterFile
	readEmbedded(t, "chapters/n5.json", &chapters)
	slugs := map[string]bool{}
	for _, c := range chapters.Chapters {
		slugs[c.Slug] = true
	}
	var file ListeningFile
	readEmbedded(t, "listening/n5.json", &file)
	if err := CheckListeningFile(&file); err != nil {
		t.Fatal(err)
	}
	for _, c := range file.Chapters {
		if !slugs[c.Slug] {
			t.Errorf("listening chapter %s is not in chapters/n5.json", c.Slug)
		}
		if len(c.Passages) == 0 {
			t.Errorf("chapter %s has no passages", c.Slug)
		}
	}
}

func readEmbedded(t *testing.T, path string, v any) {
	t.Helper()
	data, err := contentfiles.FS.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
