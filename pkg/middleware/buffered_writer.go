package middleware

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

// bufferedWriter holds the status and body a handler writes so a middleware
// can inspect or transform the whole response (compress it, tag it) before it
// goes out. API responses are small JSON documents, so buffering is cheap;
// streaming responses must not pass through it.
type bufferedWriter struct {
	gin.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *bufferedWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}

func (w *bufferedWriter) WriteHeaderNow() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
}

func (w *bufferedWriter) Write(b []byte) (int, error) {
	w.WriteHeaderNow()
	return w.body.Write(b) //nolint:wrapcheck // io.Writer pass-through
}

func (w *bufferedWriter) WriteString(s string) (int, error) {
	w.WriteHeaderNow()
	return w.body.WriteString(s) //nolint:wrapcheck // io.StringWriter pass-through
}

func (w *bufferedWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *bufferedWriter) Size() int     { return w.body.Len() }
func (w *bufferedWriter) Written() bool { return w.status != 0 }

// Flush is a no-op: nothing reaches the client until the middleware sends it.
func (w *bufferedWriter) Flush() {}

// sendRaw writes the buffered response unchanged.
func (w *bufferedWriter) sendRaw(out gin.ResponseWriter) {
	if !w.Written() {
		return
	}
	out.WriteHeader(w.status)
	_, _ = out.Write(w.body.Bytes())
}

func bodyAllowed(status int) bool {
	return status != http.StatusNoContent && status != http.StatusNotModified && status >= http.StatusOK
}
