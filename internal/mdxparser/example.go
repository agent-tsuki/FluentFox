// Package mdxparser — example.go.
// Parses <Example> JSX blocks from MDX body lines.
package mdxparser

import "strings"

// ParsedExample holds a single example extracted from the MDX body.
type ParsedExample struct {
	Japanese   string
	Romaji     string
	English    string
	ConceptRef string // title of the parent concept
}

// ExampleParser extracts Example blocks from MDX body lines.
type ExampleParser struct{}

// NewExampleParser constructs an ExampleParser.
func NewExampleParser() *ExampleParser { return &ExampleParser{} }

// Parse scans lines for <Example> blocks.
func (p *ExampleParser) Parse(lines []string) ([]ParsedExample, error) {
	var examples []ParsedExample

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "<Example") {
			continue
		}

		examples = append(examples, ParsedExample{
			Japanese:   extractAttr(line, "japanese"),
			Romaji:     extractAttr(line, "romaji"),
			English:    extractAttr(line, "english"),
			ConceptRef: extractAttr(line, "concept"),
		})
	}

	return examples, nil
}
