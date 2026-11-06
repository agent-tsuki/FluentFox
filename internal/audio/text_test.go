package audio

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type sentenceCase struct {
	Name      string   `json:"name"`
	Text      string   `json:"text"`
	Sentences []string `json:"sentences"`
}

// The same file runs in the frontend's tests, so both sides split alike.
func TestSentences(t *testing.T) {
	raw, err := os.ReadFile("testdata/sentences.json")
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	var cases []sentenceCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got := Sentences(tc.Text)
			if len(got) == 0 && len(tc.Sentences) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.Sentences) {
				t.Errorf("Sentences(%q) = %q, want %q", tc.Text, got, tc.Sentences)
			}
		})
	}
}

// The frontend test checks the same value, so both sides name a clip alike.
func TestClipHashIsSHA256OfTheSentence(t *testing.T) {
	const want = "1ba626dd871ec1ff07633de4c85693e2fed2ceb5e619601804187c86275cf2fb" // sha256("あさ")
	if got := clipHash("あさ"); got != want {
		t.Errorf("clipHash(%q) = %s, want %s", "あさ", got, want)
	}
}
