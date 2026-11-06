package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Responses smaller than this are not worth the gzip framing and CPU.
const compressMinBytes = 1024

// gzip.NewWriterLevel allocates ~800 KB of tables; pooling makes compression
// close to allocation-free per request.
var gzipWriters = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression) //nolint:errcheck // the level is valid
		return w
	},
}

// Compress gzips JSON and text responses of 1 KB or more for clients that
// accept it. The course content routes return 20-70 KB of JSON; gzip makes
// that 4-6x smaller, which matters more on a slow phone connection than any
// server-side time. Responses that already carry a Content-Encoding pass
// through untouched.
func Compress() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add, not Set: CORS has already put "Origin" in Vary.
		c.Writer.Header().Add("Vary", "Accept-Encoding")
		if c.Request.Method == http.MethodHead || !acceptsGzip(c.GetHeader("Accept-Encoding")) {
			c.Next()
			return
		}

		out := c.Writer
		buf := &bufferedWriter{ResponseWriter: out}
		c.Writer = buf
		c.Next()
		c.Writer = out

		h := out.Header()
		if buf.body.Len() < compressMinBytes || !bodyAllowed(buf.Status()) ||
			h.Get("Content-Encoding") != "" || !compressible(h.Get("Content-Type")) {
			buf.sendRaw(out)
			return
		}

		h.Set("Content-Encoding", "gzip")
		h.Del("Content-Length")
		out.WriteHeader(buf.Status())
		gz := gzipWriters.Get().(*gzip.Writer) //nolint:forcetypeassert // the pool only holds *gzip.Writer
		gz.Reset(out)
		_, _ = gz.Write(buf.body.Bytes())
		_ = gz.Close()
		gzipWriters.Put(gz)
	}
}

// acceptsGzip reports whether the Accept-Encoding header allows gzip
// (an explicit "gzip;q=0" refuses it).
func acceptsGzip(header string) bool {
	for part := range strings.SplitSeq(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(name), "gzip") && strings.TrimSpace(name) != "*" {
			continue
		}
		q := strings.ReplaceAll(params, " ", "")
		return q != "q=0" && q != "q=0.0" && q != "q=0.00" && q != "q=0.000"
	}
	return false
}

func compressible(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "json") || strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "javascript") || strings.Contains(ct, "xml")
}
