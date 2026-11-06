package middleware

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

var bigJSON = `{"data":"` + strings.Repeat("kana ", 2000) + `"}`

func newPerfRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS([]string{"https://foxsensei.com"}))
	r.Use(Compress())
	r.Use(PublicCache(PublicCacheRule{Prefix: "/content/", CacheControl: "public, max-age=300"}))
	r.Use(ServerTiming())
	r.GET("/content/kana", handler)
	r.GET("/private", handler)
	r.POST("/content/kana", handler)
	return r
}

func jsonHandler(body string, status int) gin.HandlerFunc {
	return func(c *gin.Context) { c.Data(status, "application/json", []byte(body)) }
}

func do(r http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestCompressGzipsLargeJSON(t *testing.T) {
	r := newPerfRouter(jsonHandler(bigJSON, http.StatusOK))
	rec := do(r, http.MethodGet, "/content/kana", map[string]string{"Accept-Encoding": "br, gzip", "Origin": "https://foxsensei.com"})

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	vary := strings.Join(rec.Header().Values("Vary"), ",")
	if !strings.Contains(vary, "Origin") || !strings.Contains(vary, "Accept-Encoding") {
		t.Fatalf("Vary = %q, want both Origin and Accept-Encoding", vary)
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != bigJSON {
		t.Fatal("decompressed body differs from the handler's")
	}
	if rec.Body.Len() >= len(bigJSON) {
		t.Fatalf("gzip body %d bytes is not smaller than %d", rec.Body.Len(), len(bigJSON))
	}
}

func TestCompressSkips(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		encoding string
	}{
		{"client without gzip", bigJSON, ""},
		{"client refusing gzip", bigJSON, "gzip;q=0"},
		{"small body", `{"ok":true}`, "gzip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPerfRouter(jsonHandler(tt.body, http.StatusOK))
			rec := do(r, http.MethodGet, "/private", map[string]string{"Accept-Encoding": tt.encoding})
			if rec.Header().Get("Content-Encoding") != "" {
				t.Fatal("response was compressed")
			}
			if rec.Body.String() != tt.body {
				t.Fatalf("body changed: %q", rec.Body.String())
			}
		})
	}
}

func TestPublicCacheTagsAndRevalidates(t *testing.T) {
	r := newPerfRouter(jsonHandler(bigJSON, http.StatusOK))
	first := do(r, http.MethodGet, "/content/kana", nil)
	if got := first.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("Cache-Control = %q", got)
	}
	etag := first.Header().Get("ETag")
	if !strings.HasPrefix(etag, `W/"`) {
		t.Fatalf("ETag = %q, want a weak tag", etag)
	}
	if !strings.Contains(strings.Join(first.Header().Values("Vary"), ","), "Origin") {
		t.Fatal("cacheable response without Vary: Origin")
	}

	again := do(r, http.MethodGet, "/content/kana", map[string]string{"If-None-Match": etag, "Accept-Encoding": "gzip"})
	if again.Code != http.StatusNotModified || again.Body.Len() != 0 {
		t.Fatalf("revalidation got %d with %d bytes, want an empty 304", again.Code, again.Body.Len())
	}

	stale := do(r, http.MethodGet, "/content/kana", map[string]string{"If-None-Match": `W/"other"`})
	if stale.Code != http.StatusOK || stale.Body.String() != bigJSON {
		t.Fatalf("changed content got %d, want the full 200", stale.Code)
	}
}

func TestPublicCacheLeavesOthersAlone(t *testing.T) {
	errRouter := newPerfRouter(jsonHandler(`{"error":"x"}`, http.StatusNotFound))
	if rec := do(errRouter, http.MethodGet, "/content/kana", nil); rec.Header().Get("Cache-Control") != "" || rec.Header().Get("ETag") != "" {
		t.Fatal("an error response was made cacheable")
	}
	r := newPerfRouter(jsonHandler(bigJSON, http.StatusOK))
	for _, rec := range []*httptest.ResponseRecorder{do(r, http.MethodPost, "/content/kana", nil), do(r, http.MethodGet, "/private", nil)} {
		if rec.Header().Get("Cache-Control") != "" || rec.Header().Get("ETag") != "" {
			t.Fatal("a non-public or non-GET response was made cacheable")
		}
	}
}

func TestServerTimingOnlyOnReads(t *testing.T) {
	r := newPerfRouter(jsonHandler(bigJSON, http.StatusOK))
	got := do(r, http.MethodGet, "/content/kana", nil).Header().Get("Server-Timing")
	if !strings.HasPrefix(got, "app;dur=") || !strings.Contains(got, `db;dur=0.00;desc="0 queries"`) {
		t.Fatalf("Server-Timing = %q", got)
	}
	if got := do(r, http.MethodPost, "/content/kana", nil).Header().Get("Server-Timing"); got != "" {
		t.Fatalf("POST got Server-Timing %q; writes must not expose query counts", got)
	}
}
