package emailcheck

import (
	_ "embed"
	"strings"
)

// Upstream list from github.com/disposable-email-domains (CC0).
// Refresh it with `make update-disposable-domains`.
//
//go:embed data/disposable_email_blocklist.conf
var upstreamBlocklist string

//go:embed data/custom_blocklist.txt
var customBlocklist string

//go:embed data/custom_allowlist.txt
var customAllowlist string

//go:embed data/providers.txt
var providerList string

type domainSet map[string]struct{}

func newDomainSet(sources ...string) domainSet {
	set := domainSet{}
	for _, source := range sources {
		for _, domain := range parseDomainList(source) {
			set[domain] = struct{}{}
		}
	}
	return set
}

func parseDomainList(source string) []string {
	var domains []string
	for _, line := range strings.Split(source, "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		domains = append(domains, line)
	}
	return domains
}

// Stops before the bare TLD, otherwise one bad entry like "com" would block everything.
func (s domainSet) matches(domain string) bool {
	for {
		if _, ok := s[domain]; ok {
			return true
		}
		dot := strings.Index(domain, ".")
		if dot < 0 {
			return false
		}
		parent := domain[dot+1:]
		if !strings.Contains(parent, ".") {
			return false
		}
		domain = parent
	}
}
