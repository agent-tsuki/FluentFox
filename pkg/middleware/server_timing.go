package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fluentfox/api/pkg/database"
)

// ServerTiming adds a Server-Timing header that splits each GET response's
// time into the API's own work ("app") and time spent waiting on the
// database ("db"), visible in the browser's network panel:
//
//	Server-Timing: app;dur=0.4, db;dur=12.1;desc="3 queries"
//
// It is limited to GET and HEAD: on writes such as login or sign-up, a query
// count could reveal whether an account exists.
func ServerTiming() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}
		ctx, timer := database.WithQueryTimer(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		w := &timingWriter{ResponseWriter: c.Writer, start: time.Now(), timer: timer}
		c.Writer = w
		c.Next()
		c.Writer = w.ResponseWriter
	}
}

// timingWriter sets the header just before the status line goes out, the
// last moment headers can still change.
type timingWriter struct {
	gin.ResponseWriter
	start time.Time
	timer *database.QueryTimer
	done  bool
}

func (w *timingWriter) stamp() {
	if w.done {
		return
	}
	w.done = true
	total := time.Since(w.start)
	db, queries := w.timer.Total()
	w.Header().Set("Server-Timing", fmt.Sprintf(`app;dur=%.2f, db;dur=%.2f;desc="%d queries"`,
		ms(total-db), ms(db), queries))
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func (w *timingWriter) WriteHeader(code int) {
	w.stamp()
	w.ResponseWriter.WriteHeader(code)
}

func (w *timingWriter) WriteHeaderNow() {
	w.stamp()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *timingWriter) Write(b []byte) (int, error) {
	w.stamp()
	return w.ResponseWriter.Write(b) //nolint:wrapcheck // io.Writer pass-through
}

func (w *timingWriter) WriteString(s string) (int, error) {
	w.stamp()
	return w.ResponseWriter.WriteString(s) //nolint:wrapcheck // io.StringWriter pass-through
}
