// Package mdxparser — concept.go.
// Parses <Concept> JSX blocks from MDX body lines into ParsedConcept structs.
package mdxparser

import (
	"strings"
)

// ParsedConcept holds a single grammar concept extracted from the MDX body.
type ParsedConcept struct {
	Title       string
	Explanation string
	OrderIndex  int
}

// ConceptParser extracts Concept blocks from MDX body lines.
type ConceptParser struct{}

// NewConceptParser constructs a ConceptParser.
func NewConceptParser() *ConceptParser { return &ConceptParser{} }

// Parse scans lines for <Concept> blocks and extracts title and explanation.
func (p *ConceptParser) Parse(lines []string) ([]ParsedConcept, error) {
	var concepts []ParsedConcept
	order := 0

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "<Concept") {
			continue
		}

		title := extractAttr(line, "title")
		var explanationLines []string

		for j := i + 1; j < len(lines); j++ {
			l := strings.TrimSpace(lines[j])
			if strings.HasPrefix(l, "</Concept>") || strings.HasPrefix(l, "/>") {
				i = j
				break
			}
			if l != "" {
				explanationLines = append(explanationLines, l)
			}
		}

		if title != "" {
			concepts = append(concepts, ParsedConcept{
				Title:       title,
				Explanation: strings.Join(explanationLines, " "),
				OrderIndex:  order,
			})
			order++
		}
	}

	return concepts, nil
}

// extractAttr extracts a JSX attribute value: attr="value" or attr='value'.
func extractAttr(line, attr string) string {
	search := attr + `="`
	idx := strings.Index(line, search)
	if idx == -1 {
		search = attr + `='`
		idx = strings.Index(line, search)
		if idx == -1 {
			return ""
		}
	}
	start := idx + len(search)
	end := strings.IndexAny(line[start:], `"'`)
	if end == -1 {
		return ""
	}
	return line[start : start+end]
}
