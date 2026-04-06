// Package mdxparser — vocab.go.
// Parses <Vocab> JSX blocks from MDX body lines.
package mdxparser

import "strings"

// ParsedVocab holds a vocabulary item extracted from the MDX body.
type ParsedVocab struct {
	Word      string
	Reading   string
	Meaning   string
	PartOfSpeech string
}

// VocabParser extracts Vocab blocks.
type VocabParser struct{}

// NewVocabParser constructs a VocabParser.
func NewVocabParser() *VocabParser { return &VocabParser{} }

// Parse scans lines for <Vocab> blocks.
func (p *VocabParser) Parse(lines []string) ([]ParsedVocab, error) {
	var vocab []ParsedVocab

	for _, line := range lines {
		l := strings.TrimSpace(line)
		if !strings.HasPrefix(l, "<Vocab") {
			continue
		}
		vocab = append(vocab, ParsedVocab{
			Word:         extractAttr(l, "word"),
			Reading:      extractAttr(l, "reading"),
			Meaning:      extractAttr(l, "meaning"),
			PartOfSpeech: extractAttr(l, "pos"),
		})
	}

	return vocab, nil
}
