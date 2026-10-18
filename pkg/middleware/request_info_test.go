package middleware

import (
	"net/netip"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRequestInfo_IPPrefix(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{name: "IPv4 is masked to /24", ip: "203.0.113.77", want: "203.0.113.0/24"},
		{name: "IPv6 is masked to /48", ip: "2001:db8:abcd:12::1", want: "2001:db8:abcd::/48"},
		{name: "IPv4-mapped IPv6 is treated as IPv4", ip: "::ffff:198.51.100.9", want: "198.51.100.0/24"},
		{name: "unknown address gives an empty prefix", ip: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info RequestInfo
			if tt.ip != "" {
				info.IP = netip.MustParseAddr(tt.ip).Unmap()
			}
			if got := info.IPPrefix(); got != tt.want {
				t.Errorf("IPPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		max     int
		wantLen int
	}{
		{name: "short string is unchanged", in: "Mozilla/5.0", max: 512, wantLen: len("Mozilla/5.0")},
		{name: "long string is cut to max bytes", in: strings.Repeat("a", 600), max: 512, wantLen: 512},
		{name: "a split multi-byte character is dropped", in: "ab日本", max: 4, wantLen: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.in, tt.max)
			if len(got) != tt.wantLen || !utf8.ValidString(got) {
				t.Errorf("truncate() = %q (len %d), want len %d and valid UTF-8", got, len(got), tt.wantLen)
			}
		})
	}
}

func TestRequestInfo_IsPublic(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public IPv4", ip: "203.0.113.7", want: true},
		{name: "public IPv6", ip: "2001:4860:4860::8888", want: true},
		{name: "private LAN", ip: "192.168.1.10", want: false},
		{name: "loopback", ip: "127.0.0.1", want: false},
		{name: "proxy shared address space", ip: "100.64.0.3", want: false},
		{name: "unknown", ip: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info RequestInfo
			if tt.ip != "" {
				info.IP = netip.MustParseAddr(tt.ip)
			}
			if got := info.IsPublic(); got != tt.want {
				t.Errorf("IsPublic(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
