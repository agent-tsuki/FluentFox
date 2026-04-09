// Package mdxparser — frontmatter.go.
// Extracts and parses YAML frontmatter (between --- delimiters) from MDX files.
package mdxparser

import (
	"fmt"
	"strings"
)

// FrontmatterData holds the parsed fields from the --- block.
type FrontmatterData struct {
	Title       string
	Slug        string
	JLPTLevel   string
	OrderIndex  int
	Description string
	Published   bool
}

// FrontmatterParser extracts frontmatter from the top of an MDX file.
type FrontmatterParser struct{}

// NewFrontmatterParser constructs a FrontmatterParser.
func NewFrontmatterParser() *FrontmatterParser { return &FrontmatterParser{} }

// Extract separates frontmatter from body lines.
// Returns the parsed FrontmatterData and the remaining body lines.
func (p *FrontmatterParser) Extract(lines []string) (FrontmatterData, []string, error) {
	if len(lines) == 0 || lines[0] != "---" {
		return FrontmatterData{}, lines, nil
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return FrontmatterData{}, nil, fmt.Errorf("frontmatter: unclosed --- block")
	}

	fm := FrontmatterData{Published: true}
	for _, line := range lines[1:end] {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "title":
			fm.Title = value
		case "slug":
			fm.Slug = value
		case "jlpt_level":
			fm.JLPTLevel = value
		case "description":
			fm.Description = value
		case "published":
			fm.Published = value == "true"
		}
	}

	if fm.Title == "" || fm.Slug == "" {
		return FrontmatterData{}, nil, fmt.Errorf("frontmatter: missing required field title or slug")
	}

	return fm, lines[end+1:], nil
}
