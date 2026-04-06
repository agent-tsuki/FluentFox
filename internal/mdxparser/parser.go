// Package mdxparser parses MDX grammar files into structured data
// ready for database insertion. It is used exclusively by cmd/sync-content.
// It must never be imported by any HTTP handler or service.
package mdxparser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParsedChapter holds all structured data extracted from a single MDX file.
type ParsedChapter struct {
	Frontmatter FrontmatterData
	Concepts    []ParsedConcept
	Vocab       []ParsedVocab
	Examples    []ParsedExample
	Sidebar     *ParsedSidebar
}

// Parser orchestrates parsing of an MDX file into a ParsedChapter.
type Parser struct {
	fmParser   *FrontmatterParser
	conceptParser *ConceptParser
	vocabParser   *VocabParser
	exampleParser *ExampleParser
	sidebarParser *SidebarParser
	cleaner       *Cleaner
}

// New constructs a Parser with all sub-parsers initialised.
func New() *Parser {
	return &Parser{
		fmParser:      NewFrontmatterParser(),
		conceptParser: NewConceptParser(),
		vocabParser:   NewVocabParser(),
		exampleParser: NewExampleParser(),
		sidebarParser: NewSidebarParser(),
		cleaner:       NewCleaner(),
	}
}

// ParseFile reads an MDX file at path and returns the structured ParsedChapter.
func (p *Parser) ParseFile(path string) (*ParsedChapter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("mdxparser: open %q: %w", path, err)
	}
	defer f.Close()

	lines, err := readLines(f)
	if err != nil {
		return nil, fmt.Errorf("mdxparser: read lines %q: %w", path, err)
	}

	lines = p.cleaner.Strip(lines)

	fm, body, err := p.fmParser.Extract(lines)
	if err != nil {
		return nil, fmt.Errorf("mdxparser: frontmatter %q: %w", path, err)
	}

	concepts, err := p.conceptParser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("mdxparser: concepts %q: %w", path, err)
	}

	vocab, err := p.vocabParser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("mdxparser: vocab %q: %w", path, err)
	}

	examples, err := p.exampleParser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("mdxparser: examples %q: %w", path, err)
	}

	sidebar, err := p.sidebarParser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("mdxparser: sidebar %q: %w", path, err)
	}

	return &ParsedChapter{
		Frontmatter: fm,
		Concepts:    concepts,
		Vocab:       vocab,
		Examples:    examples,
		Sidebar:     sidebar,
	}, nil
}

// ParseDirectory walks a directory and parses every .mdx file found.
func (p *Parser) ParseDirectory(dir string) ([]*ParsedChapter, error) {
	var chapters []*ParsedChapter

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".mdx") {
			return nil
		}
		ch, err := p.ParseFile(path)
		if err != nil {
			return fmt.Errorf("mdxparser: walk: %w", err)
		}
		chapters = append(chapters, ch)
		return nil
	})

	return chapters, err
}

func readLines(f *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
