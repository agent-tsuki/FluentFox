package learn

import (
	"regexp"
	"slices"
	"strings"

	"github.com/fluentfox/api/internal/common"
)

// answerKind is what a question shows or asks for: an item's romaji,
// character, first meaning or key reading.
type answerKind int

const (
	kindNone answerKind = iota
	kindRomaji
	kindChar
	kindMeaning
	kindReading
)

var questionAnswerKinds = map[common.QuestionType]answerKind{
	common.QuestionKanaMCQRomaji:     kindRomaji,
	common.QuestionKanaMCQKana:       kindChar,
	common.QuestionKanaTyped:         kindRomaji,
	common.QuestionKanaListen:        kindChar,
	common.QuestionKanaBuild:         kindChar,
	common.QuestionKanaLookalike:     kindChar,
	common.QuestionKanaSpeed:         kindRomaji,
	common.QuestionKanaMatch:         kindChar,
	common.QuestionKanaTrace:         kindNone,
	common.QuestionKanjiMeaningMCQ:   kindMeaning,
	common.QuestionKanjiMeaningTyped: kindMeaning,
	common.QuestionKanjiReadingMCQ:   kindReading,
	common.QuestionKanjiReadingTyped: kindReading,
	common.QuestionKanjiListen:       kindChar,
	common.QuestionKanjiLookalike:    kindChar,
}

// Listening prompts are spoken by the client: the kana itself, or the
// kanji's key reading.
var questionPromptKinds = map[common.QuestionType]answerKind{
	common.QuestionKanaMCQRomaji:     kindChar,
	common.QuestionKanaMCQKana:       kindRomaji,
	common.QuestionKanaTyped:         kindChar,
	common.QuestionKanaListen:        kindChar,
	common.QuestionKanaBuild:         kindRomaji,
	common.QuestionKanaLookalike:     kindRomaji,
	common.QuestionKanaSpeed:         kindChar,
	common.QuestionKanaMatch:         kindRomaji,
	common.QuestionKanaTrace:         kindChar,
	common.QuestionKanjiMeaningMCQ:   kindChar,
	common.QuestionKanjiMeaningTyped: kindChar,
	common.QuestionKanjiReadingMCQ:   kindChar,
	common.QuestionKanjiReadingTyped: kindChar,
	common.QuestionKanjiListen:       kindReading,
	common.QuestionKanjiLookalike:    kindMeaning,
}

func itemValue(item *Item, kind answerKind) string {
	switch kind {
	case kindRomaji:
		return item.Romaji
	case kindChar:
		return item.Char
	case kindMeaning:
		if len(item.Meanings) > 0 {
			return item.Meanings[0]
		}
		return ""
	case kindReading:
		return item.KeyReading
	default:
		return ""
	}
}

// acceptedValues are the answers accepted besides itemValue.
func acceptedValues(item *Item, kind answerKind) []string {
	main := itemValue(item, kind)
	var all []string
	switch kind {
	case kindRomaji:
		all = item.AltRomaji
	case kindMeaning:
		all = slices.Concat(item.Meanings, item.Synonyms)
	case kindReading:
		all = item.Readings
	default:
		return nil
	}
	out := make([]string, 0, len(all))
	for _, v := range uniqueStrings(all) {
		if v != main {
			out = append(out, v)
		}
	}
	return out
}

// sameAnswer reports whether other would also be a right answer for target,
// such as じ and ぢ (both "ji") or two kanji that share a meaning.
func sameAnswer(facet common.CardFacet, target, other *Item) bool {
	switch facet {
	case common.CardFacetMeaning:
		return overlaps(meaningSet(target), meaningSet(other))
	case common.CardFacetReading:
		return overlaps(readingSet(target), readingSet(other))
	default:
		return overlaps(romajiSet(target), romajiSet(other))
	}
}

func romajiSet(item *Item) []string {
	out := []string{normalizeRomaji(item.Romaji)}
	for _, r := range item.AltRomaji {
		out = append(out, normalizeRomaji(r))
	}
	return out
}

func meaningSet(item *Item) []string {
	all := slices.Concat(item.Meanings, item.Synonyms)
	out := make([]string, 0, len(all))
	for _, m := range all {
		out = append(out, normalizeMeaningAnswer(m))
	}
	return out
}

func readingSet(item *Item) []string {
	out := []string{normalizeReading(item.KeyReading)}
	for _, r := range item.Readings {
		out = append(out, normalizeReading(r))
	}
	return out
}

func overlaps(a, b []string) bool {
	for _, x := range a {
		if x != "" && slices.Contains(b, x) {
			return true
		}
	}
	return false
}

func normalizeRomaji(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

var (
	meaningBrackets    = regexp.MustCompile(`\([^)]*\)`)
	meaningPunctuation = regexp.MustCompile(`[^a-z0-9\s'-]`)
	meaningArticle     = regexp.MustCompile(`^\s*(to|a|an|the)\s+`)
)

// normalizeMeaningAnswer matches the client's normaliseMeaning, so a typed
// answer the client marked right is not marked wrong here: no brackets,
// punctuation, leading "to" or article.
func normalizeMeaningAnswer(s string) string {
	s = strings.ToLower(s)
	s = meaningBrackets.ReplaceAllString(s, "")
	s = meaningPunctuation.ReplaceAllString(s, " ")
	s = meaningArticle.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

const (
	katakanaFirst      = 'ァ'
	katakanaLast       = 'ヶ'
	katakanaToHiragana = 'ァ' - 'ぁ'
)

// On'yomi are stored in katakana and kun'yomi in hiragana; answers compare
// in hiragana, without the separators the client also strips.
func normalizeReading(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case strings.ContainsRune(" .-ー・()", r):
			continue
		case r >= katakanaFirst && r <= katakanaLast:
			b.WriteRune(r - katakanaToHiragana)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
