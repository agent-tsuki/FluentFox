package middleware

import (
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

func TestRequestInfoCapture_ClientIP(t *testing.T) {
	tests := []struct {
		name              string
		trustVercelHeader bool
		vercelHeader      string
		remoteAddr        string
		want              string
	}{
		{
			name:              "uses Vercel's client IP instead of the function peer",
			trustVercelHeader: true,
			vercelHeader:      "203.0.113.77",
			remoteAddr:        "152.233.15.9:443",
			want:              "203.0.113.77",
		},
		{
			name:         "does not trust a Vercel header outside Vercel",
			vercelHeader: "203.0.113.77",
			remoteAddr:   "198.51.100.9:443",
			want:         "198.51.100.9",
		},
		{
			name:              "falls back when Vercel header is invalid",
			trustVercelHeader: true,
			vercelHeader:      "203.0.113.77, 152.233.15.9",
			remoteAddr:        "152.233.15.9:443",
			want:              "152.233.15.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			if err := r.SetTrustedProxies(nil); err != nil {
				t.Fatal(err)
			}
			r.Use(RequestInfoCapture(tt.trustVercelHeader))

			var got RequestInfo
			r.GET("/", func(c *gin.Context) {
				got = RequestInfoFromContext(c.Request.Context())
			})

			req := httptest.NewRequest("GET", "http://example.test/", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set(vercelForwardedForHeader, tt.vercelHeader)
			r.ServeHTTP(httptest.NewRecorder(), req)

			if value := got.IP.String(); value != tt.want {
				t.Errorf("captured IP = %q, want %q", value, tt.want)
			}
		})
	}
}

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
