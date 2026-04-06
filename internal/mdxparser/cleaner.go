// Package mdxparser — cleaner.go.
// Strips MDX import statements, export declarations, and blank-line runs
// before the structural parsers see the file content.
package mdxparser

import "strings"

// Cleaner removes non-content lines from MDX source before parsing.
type Cleaner struct{}

// NewCleaner constructs a Cleaner.
func NewCleaner() *Cleaner { return &Cleaner{} }

// Strip removes import/export lines and normalises blank line runs to a single blank.
func (c *Cleaner) Strip(lines []string) []string {
	var result []string
	prevBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Remove MDX imports and exports.
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export ") {
			continue
		}

		// Collapse consecutive blank lines.
		if trimmed == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
		} else {
			prevBlank = false
		}

		result = append(result, line)
	}

	return result
}
