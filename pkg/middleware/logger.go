// Package middleware — logger.go
// Generates a UUID request_id per request, injects it into the stdlib context,
// and logs method, path, status, and latency using zap after the handler returns.
package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
)

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerFromContext returns the per-request zap logger, falling back to base.
func LoggerFromContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return l
	}
	return base
}

// Logger attaches a request_id and enriched logger to every request,
// then logs the completed request after the handler chain returns.
func Logger(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewString()
		start := time.Now()

		reqLogger := base.With(zap.String("request_id", requestID))

		ctx := context.WithValue(c.Request.Context(), requestIDKey, requestID)
		ctx = context.WithValue(ctx, loggerKey, reqLogger)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", requestID)

		c.Next()

		reqLogger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("remote_addr", c.ClientIP()),
		)
	}
}
