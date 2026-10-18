package humautil

import (
	"errors"
	"net/http"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/fluentfox/api/pkg/exceptions"
)

func TestMapErr_LogsServerErrorsForAlerts(t *testing.T) {
	cause := errors.New("db down")
	tests := []struct {
		name      string
		err       error
		wantLog   bool
		wantError string
	}{
		{name: "plain error is logged", err: cause, wantLog: true, wantError: "db down"},
		{name: "500 app error logs its cause", err: exceptions.Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "try again", cause), wantLog: true, wantError: "db down"},
		{name: "503 app error without cause logs its message", err: exceptions.New(http.StatusServiceUnavailable, "EMAIL_SEND_FAILED", "could not send"), wantLog: true, wantError: "could not send"},
		{name: "4xx app error is not logged", err: exceptions.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "bad login"), wantLog: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.ErrorLevel)
			_ = MapErr(tt.err, zap.New(core))

			entries := logs.FilterMessage(InternalErrorLogMessage).All()
			if !tt.wantLog {
				if len(entries) != 0 {
					t.Fatalf("expected no log, got %d", len(entries))
				}
				return
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 log, got %d", len(entries))
			}
			if got := entries[0].ContextMap()["error"]; got != tt.wantError {
				t.Errorf("logged error = %v, want %q", got, tt.wantError)
			}
		})
	}
}
