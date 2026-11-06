package audio

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Sentences splits text into the units that are spoken and stored as one clip
// each, so a sentence shared by two paragraphs is generated once. The frontend
// (src/lib/audio/speechText.ts) runs the same steps to find a clip on the CDN
// without calling the API; testdata/sentences.json is shared by both test
// suites and must pass in each.
//
// Steps: NFKC (half-width kana and full-width ASCII fold, so ！ becomes !),
// drop U+FEFF, end a sentence after 。 ! ? (keeping any closing brackets or
// quotes and further marks that follow, as in 。」 or !?) and at line breaks,
// collapse runs of whitespace to one space and trim. A piece with no letter or
// digit is joined to the sentence before it.
func Sentences(text string) []string {
	folded := strings.ReplaceAll(norm.NFKC.String(text), "\uFEFF", "")
	var out []string
	var current strings.Builder
	flush := func() {
		piece := strings.Join(strings.FieldsFunc(current.String(), isSpace), " ")
		current.Reset()
		switch {
		case piece == "":
		case !hasLetterOrDigit(piece) && len(out) > 0:
			out[len(out)-1] += piece
		default:
			out = append(out, piece)
		}
	}
	ending := false
	for _, r := range folded {
		if r == '\n' || r == '\r' {
			ending = false
			flush()
			continue
		}
		if ending && !isSentenceEnd(r) && !isClosing(r) {
			ending = false
			flush()
		}
		current.WriteRune(r)
		if isSentenceEnd(r) {
			ending = true
		}
	}
	flush()
	return out
}

func isSentenceEnd(r rune) bool {
	return r == '。' || r == '!' || r == '?'
}

// Closing brackets and final quotes: 」』）】 and ” ’.
func isClosing(r rune) bool {
	return unicode.In(r, unicode.Pe, unicode.Pf)
}

// Go's unicode.IsSpace and JavaScript's \s differ only in U+0085 (and U+FEFF,
// removed above); both sides count U+0085 as a space.
func isSpace(r rune) bool {
	return unicode.IsSpace(r)
}

func hasLetterOrDigit(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

// clipHash names a sentence's clip. The voice is part of the object path, not
// the hash, so the frontend needs only the text and the voice's base URL.
func clipHash(sentence string) string {
	sum := sha256.Sum256([]byte(sentence))
	return hex.EncodeToString(sum[:])
}
