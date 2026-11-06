package middleware

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PublicCacheRule sets Cache-Control for successful GETs whose path starts
// with Prefix.
type PublicCacheRule struct {
	Prefix       string
	CacheControl string
}

// PublicCache lets browsers and a CDN reuse public, user-independent
// responses (course content, the class catalogue):
//
//   - Cache-Control from the first matching rule, so a repeat visit or an edge
//     cache answers without reaching the API at all;
//   - a weak ETag over the body, so once that expires the browser revalidates
//     with If-None-Match and gets a body-less 304 when nothing changed.
//
// Only 200 responses to GET are tagged; errors and other methods pass through.
func PublicCache(rules ...PublicCacheRule) gin.HandlerFunc {
	return func(c *gin.Context) {
		rule, ok := matchCacheRule(rules, c.Request.URL.Path)
		if !ok || c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		out := c.Writer
		buf := &bufferedWriter{ResponseWriter: out}
		c.Writer = buf
		c.Next()
		c.Writer = out

		if buf.Status() != http.StatusOK {
			buf.sendRaw(out)
			return
		}
		etag := weakETag(buf.body.Bytes())
		h := out.Header()
		h.Set("Cache-Control", rule.CacheControl)
		// CORS only adds Vary: Origin for allowed origins. A shared cache must
		// also keep the copy without Access-Control-Allow-Origin apart.
		if !varyHas(h.Values("Vary"), "Origin") {
			h.Add("Vary", "Origin")
		}
		h.Set("ETag", etag)
		if etagMatches(c.GetHeader("If-None-Match"), etag) {
			h.Del("Content-Type")
			h.Del("Content-Length")
			out.WriteHeader(http.StatusNotModified)
			return
		}
		buf.sendRaw(out)
	}
}

func varyHas(values []string, name string) bool {
	for _, v := range values {
		for part := range strings.SplitSeq(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), name) {
				return true
			}
		}
	}
	return false
}

func matchCacheRule(rules []PublicCacheRule, path string) (PublicCacheRule, bool) {
	for _, r := range rules {
		if strings.HasPrefix(path, r.Prefix) {
			return r, true
		}
	}
	return PublicCacheRule{}, false
}

// FNV-1a is enough here: the tag only has to change when the body changes.
func weakETag(body []byte) string {
	h := fnv.New64a()
	_, _ = h.Write(body)
	return fmt.Sprintf(`W/"%x-%x"`, h.Sum64(), len(body))
}

func etagMatches(ifNoneMatch, etag string) bool {
	for candidate := range strings.SplitSeq(ifNoneMatch, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == strings.TrimPrefix(etag, "W/") {
			return true
		}
	}
	return false
}
