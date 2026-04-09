// Package mdxparser — sidebar.go.
// Parses optional <Sidebar> JSX blocks which contain cultural notes or tips.
package mdxparser

import "strings"

// ParsedSidebar holds a cultural note or tip block from the MDX body.
type ParsedSidebar struct {
	Title string
	Body  string
}

// SidebarParser extracts Sidebar blocks.
type SidebarParser struct{}

// NewSidebarParser constructs a SidebarParser.
func NewSidebarParser() *SidebarParser { return &SidebarParser{} }

// Parse extracts the first <Sidebar> block found in the body lines.
func (p *SidebarParser) Parse(lines []string) (*ParsedSidebar, error) {
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "<Sidebar") {
			continue
		}

		title := extractAttr(line, "title")
		var bodyLines []string

		for j := i + 1; j < len(lines); j++ {
			l := strings.TrimSpace(lines[j])
			if strings.HasPrefix(l, "</Sidebar>") {
				break
			}
			if l != "" {
				bodyLines = append(bodyLines, l)
			}
		}

		return &ParsedSidebar{
			Title: title,
			Body:  strings.Join(bodyLines, " "),
		}, nil
	}
	return nil, nil
}
