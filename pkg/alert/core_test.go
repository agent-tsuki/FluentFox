package alert

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	watchedMessage = "unhandled internal error"
	waitForAlert   = time.Second
	noAlertWindow  = 50 * time.Millisecond
)

type fakeNotifier struct {
	sent chan ServerError
}

func (f *fakeNotifier) NotifyServerError(_ context.Context, e *ServerError) error {
	f.sent <- *e
	return nil
}

func newTestLogger(interval time.Duration) (*zap.Logger, *fakeNotifier) {
	notifier := &fakeNotifier{sent: make(chan ServerError, 10)}
	core := NewCore(notifier, Config{
		Messages: []string{watchedMessage},
		Interval: interval,
		Timeout:  time.Second,
	}, zap.NewNop())
	return zap.New(core), notifier
}

func receive(t *testing.T, n *fakeNotifier) ServerError {
	t.Helper()
	select {
	case e := <-n.sent:
		return e
	case <-time.After(waitForAlert):
		t.Fatal("expected an alert, got none")
		return ServerError{}
	}
}

func expectNone(t *testing.T, n *fakeNotifier) {
	t.Helper()
	select {
	case e := <-n.sent:
		t.Fatalf("expected no alert, got %+v", e)
	case <-time.After(noAlertWindow):
	}
}

func TestCore_SendsOnlyWatchedErrors(t *testing.T) {
	tests := []struct {
		name    string
		level   zapcore.Level
		message string
		alerts  bool
	}{
		{name: "watched message at error level alerts", level: zapcore.ErrorLevel, message: watchedMessage, alerts: true},
		{name: "other error message is ignored", level: zapcore.ErrorLevel, message: "failed to send verification email", alerts: false},
		{name: "watched message at warn level is ignored", level: zapcore.WarnLevel, message: watchedMessage, alerts: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, notifier := newTestLogger(time.Minute)
			log.Log(tt.level, tt.message, zap.Error(errors.New("boom")))
			if tt.alerts {
				receive(t, notifier)
			} else {
				expectNone(t, notifier)
			}
		})
	}
}

func TestCore_CarriesRequestIDAndError(t *testing.T) {
	log, notifier := newTestLogger(time.Minute)

	log.With(zap.String("request_id", "req-123")).Error(watchedMessage, zap.Error(errors.New("login: load user: no table")))

	got := receive(t, notifier)
	if got.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", got.RequestID, "req-123")
	}
	if got.Error != "login: load user: no table" {
		t.Errorf("Error = %q, want the logged error", got.Error)
	}
	if got.Message != watchedMessage {
		t.Errorf("Message = %q, want %q", got.Message, watchedMessage)
	}
}

func TestCore_ThrottlesAndCountsSkippedAlerts(t *testing.T) {
	tests := []struct {
		name           string
		interval       time.Duration
		logged         int
		wantAlerts     int
		wantSuppressed int
	}{
		{name: "burst inside the interval sends one alert", interval: time.Hour, logged: 3, wantAlerts: 1},
		{name: "next alert after the interval reports skipped errors", interval: noAlertWindow, logged: 3, wantAlerts: 2, wantSuppressed: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, notifier := newTestLogger(tt.interval)
			for range tt.logged {
				log.Error(watchedMessage)
			}
			receive(t, notifier)
			if tt.wantAlerts == 1 {
				expectNone(t, notifier)
				return
			}

			time.Sleep(tt.interval)
			log.Error(watchedMessage)
			if got := receive(t, notifier); got.Suppressed != tt.wantSuppressed {
				t.Errorf("Suppressed = %d, want %d", got.Suppressed, tt.wantSuppressed)
			}
		})
	}
}
