package emailcheck

import "strings"

const (
	// maxTypoEdits is one for every length. Allowing two flagged real regional
	// domains such as yahoo.co.jp (two edits from yahoo.co.in).
	maxTypoEdits = 1
	// minTargetLabelLen keeps short providers such as live.com out of typo
	// matching, so real domains like hive.com are not flagged.
	minTargetLabelLen = 5
)

func (c *Checker) typoSuggestion(domain string) (string, bool) {
	if _, ok := c.providerOf[domain]; ok {
		return "", false
	}

	best, bestDistance := "", maxTypoEdits+1
	for _, provider := range c.providers {
		if firstLabelLen(provider) < minTargetLabelLen {
			continue
		}
		if d := osaDistance(domain, provider); d > 0 && d < bestDistance {
			best, bestDistance = provider, d
		}
	}
	return best, best != ""
}

func firstLabelLen(domain string) int {
	if dot := strings.Index(domain, "."); dot >= 0 {
		return dot
	}
	return len(domain)
}

// Counting a swap of neighbouring letters as one edit is what makes "gmial"
// one step from "gmail".
func osaDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	rows, cols := len(ra)+1, len(rb)+1

	d := make([][]int, rows)
	for i := range d {
		d[i] = make([]int, cols)
		d[i][0] = i
	}
	for j := range cols {
		d[0][j] = j
	}

	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[rows-1][cols-1]
}
