// Package mdxparser — segment.go.
// Defines ParsedSegment, which represents an annotated portion of an example sentence
// (e.g., kanji with furigana, grammar marker, translation segment).
package mdxparser

// SegmentType classifies what kind of annotation a segment carries.
type SegmentType string

const (
	SegmentKanji   SegmentType = "kanji"
	SegmentKana    SegmentType = "kana"
	SegmentGrammar SegmentType = "grammar"
	SegmentOther   SegmentType = "other"
)

// ParsedSegment is a single annotated chunk of a Japanese sentence.
type ParsedSegment struct {
	Text        string
	Furigana    string
	SegmentType SegmentType
	GlossEn     string // English gloss for this segment only
}
