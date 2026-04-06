// Package middleware provides HTTP middleware for chi.
// logger.go generates a UUID request_id per request, injects it into context,
// and logs method, path, status, latency, and request_id using zap.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
	userIDKey    contextKey = "user_id"
)

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerFromContext extracts the enriched zap logger from the context.
// Falls back to the base logger if none is set.
func LoggerFromContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return l
	}
	return base
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	written bool
}

func (rw *responseWriter) WriteHeader(status int) {
	if !rw.written {
		rw.status = status
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(status)
}

// Logger returns a middleware that:
//   - Generates a UUID request_id and injects it into the context.
//   - Attaches an enriched zap.Logger (with request_id field) to the context.
//   - Logs method, path, status, latency, and request_id after the handler returns.
func Logger(base *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.NewString()
			start := time.Now()

			reqLogger := base.With(zap.String("request_id", requestID))

			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			ctx = context.WithValue(ctx, loggerKey, reqLogger)

			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			reqLogger.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", wrapped.status),
				zap.Duration("latency", time.Since(start)),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}
