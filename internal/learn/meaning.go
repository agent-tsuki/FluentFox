package learn

import (
	"strings"
	"unicode/utf8"

	"github.com/fluentfox/api/internal/common"
)

// TypoDistance is zero for an exact match.
type MeaningMatch struct {
	Correct      bool
	TypoDistance int
}

// MatchMeaning checks a typed English meaning. A near miss is accepted as a
// typo, unless it is at least as close to another kanji's meaning: then the
// user may really have confused the two kanji.
func MatchMeaning(answer string, accepted, otherMeanings []string) MeaningMatch {
	a := normalizeMeaning(answer)
	if a == "" {
		return MeaningMatch{}
	}
	for _, m := range accepted {
		if normalizeMeaning(m) == a {
			return MeaningMatch{Correct: true}
		}
	}

	best, ok := closestWithinAllowance(a, accepted)
	if !ok {
		return MeaningMatch{}
	}
	for _, m := range otherMeanings {
		if Levenshtein(a, normalizeMeaning(m)) <= best {
			return MeaningMatch{}
		}
	}
	return MeaningMatch{Correct: true, TypoDistance: best}
}

func MeaningTypoAllowance(expected string) int {
	n := utf8.RuneCountInString(normalizeMeaning(expected))
	switch {
	case n <= common.MEANING_TYPO_SHORT:
		return 0
	case n <= common.MEANING_TYPO_MEDIUM:
		return common.MEANING_TYPO_MEDIUM_EDITS
	default:
		return common.MEANING_TYPO_LONG_EDITS
	}
}

// Levenshtein counts single-rune inserts, deletes and substitutions.
func Levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func closestWithinAllowance(answer string, accepted []string) (int, bool) {
	best, found := 0, false
	for _, m := range accepted {
		d := Levenshtein(answer, normalizeMeaning(m))
		if d <= MeaningTypoAllowance(m) && (!found || d < best) {
			best, found = d, true
		}
	}
	return best, found
}

func normalizeMeaning(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
