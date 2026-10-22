package middleware

import (
	"context"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
)

const requestInfoKey contextKey = "request_info"

const vercelForwardedForHeader = "X-Vercel-Forwarded-For"

// Masking keeps enough of an address to show "203.0.113.x" on a device list
// without storing the exact address of a person's home connection.
const (
	ipv4PrefixBits = 24
	ipv6PrefixBits = 48
	userAgentMax   = 512
)

// RequestInfo is the client side of a request. IP is the full address and must
// not be stored or logged; use IPPrefix for that.
type RequestInfo struct {
	IP        netip.Addr
	UserAgent string
}

// RequestInfoCapture records the address of the browser that made the
// request. On a Vercel deployment, the runtime peer is a Vercel address, so
// use Vercel's platform-controlled client-IP header instead. Everywhere else,
// Gin only trusts X-Forwarded-For from TRUSTED_PROXIES, so a client cannot
// fake it.
//
// trustVercelForwardedFor must only be true when Vercel set the request header
// (Config derives it from Vercel's VERCEL=1 runtime variable).
func RequestInfoCapture(trustVercelForwardedFor bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		info := RequestInfo{UserAgent: truncate(c.Request.UserAgent(), userAgentMax)}
		if ip, ok := requestIP(c, trustVercelForwardedFor); ok {
			info.IP = ip
		}
		c.Request = c.Request.WithContext(ContextWithRequestInfo(c.Request.Context(), info))
		c.Next()
	}
}

func requestIP(c *gin.Context, trustVercelForwardedFor bool) (netip.Addr, bool) {
	if trustVercelForwardedFor {
		// Vercel documents this as the requester's public address. Reject a
		// comma-separated chain instead of guessing which proxy supplied it.
		if ip, err := netip.ParseAddr(strings.TrimSpace(c.GetHeader(vercelForwardedForHeader))); err == nil {
			return ip.Unmap(), true
		}
	}
	if ip, err := netip.ParseAddr(c.ClientIP()); err == nil {
		return ip.Unmap(), true
	}
	return netip.Addr{}, false
}

func ContextWithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey, info)
}

func RequestInfoFromContext(ctx context.Context) RequestInfo {
	if info, ok := ctx.Value(requestInfoKey).(RequestInfo); ok {
		return info
	}
	return RequestInfo{}
}

// Railway, Docker and most clouds use shared address space (RFC 6598) between
// their proxy and the app; seeing it means TRUSTED_PROXIES is not set.
var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

// IsPublic reports whether IP is a real internet address worth looking up.
func (r RequestInfo) IsPublic() bool {
	ip := r.IP
	return ip.IsValid() && ip.IsGlobalUnicast() && !ip.IsPrivate() && !sharedAddressSpace.Contains(ip)
}

// IPPrefix is the masked network, e.g. "203.0.113.0/24", or "" when unknown.
func (r RequestInfo) IPPrefix() string {
	if !r.IP.IsValid() {
		return ""
	}
	bits := ipv6PrefixBits
	if r.IP.Is4() {
		bits = ipv4PrefixBits
	}
	prefix, err := r.IP.Prefix(bits)
	if err != nil {
		return ""
	}
	return prefix.String()
}

// Cutting at a byte count can split a multi-byte character, and Postgres
// rejects invalid UTF-8, so a split character is dropped.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.ToValidUTF8(s[:max], "")
}
