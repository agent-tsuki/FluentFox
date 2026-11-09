package learn

import (
	"slices"
	"strings"
)

// SentenceItem is a grammar practice sentence: Item.Char holds it with {}
// where the answer goes.
type SentenceItem struct {
	Answer      string
	Distractors []string
	// Other correct answers for the gap, e.g. に besides へ.
	AlsoAccept []string
	// Typed romaji the client's kana conversion cannot reach, e.g. "wa" for は.
	Romaji      []string
	Translation string
	Explanation string
}

const sentenceGap = "{}"

// Filled is the sentence as it is read aloud or translated.
func (s *SentenceItem) Filled(sentence string) string {
	return strings.Replace(sentence, sentenceGap, s.Answer, 1)
}

// gapOptions are the sentence's own choices for the gap: the answer and its
// written wrong answers (not other items, which would not fit the sentence).
func (s *SentenceItem) gapOptions() []string {
	return uniqueStrings(slices.Concat([]string{s.Answer}, s.Distractors))
}

// Particles are written は, へ and を but said wa, e and o; the client turns
// typed romaji into kana, so "wa" arrives as わ.
var particleSounds = strings.NewReplacer("は", "わ", "へ", "え", "を", "お")

const gapPunctuation = "。、「」！？!?"

func normalizeGap(s string) string {
	s = normalizeReading(s)
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(gapPunctuation, r) {
			return -1
		}
		return r
	}, s)
}

// matchesGap accepts the answer or another listed answer, typed in kana or
// kanji, the spoken form of a particle, or a listed romaji spelling.
func (s *SentenceItem) matchesGap(given string) bool {
	if slices.Contains(s.Romaji, normalizeRomaji(given)) {
		return true
	}
	typed := normalizeGap(given)
	if typed == "" {
		return false
	}
	for _, a := range slices.Concat([]string{s.Answer}, s.AlsoAccept) {
		want := normalizeGap(a)
		if typed == want || typed == particleSounds.Replace(want) {
			return true
		}
	}
	return false
}
