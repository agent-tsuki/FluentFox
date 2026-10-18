// Package alert notifies someone when the logger records a server error, so a
// 500 is seen without anyone watching the logs.
package alert

import (
	"context"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Must match the field names middleware.Logger and zap.Error write.
const (
	requestIDField = "request_id"
	errorField     = "error"
)

type ServerError struct {
	Time      time.Time
	Message   string
	RequestID string
	Error     string
	Caller    string
	Stack     string
	// Errors the throttle held back since the previous alert.
	Suppressed int
}

type Notifier interface {
	NotifyServerError(ctx context.Context, e *ServerError) error
}

type Config struct {
	// Log messages that mean a request failed with a 500.
	Messages []string
	// At most one alert is sent per Interval.
	Interval time.Duration
	// Upper bound for one send.
	Timeout time.Duration
}

// Core is a zapcore.Core to tee next to the normal one. It only reacts to
// error entries whose message is in Config.Messages.
type Core struct {
	fields     []zapcore.Field
	dispatcher *dispatcher
}

// log reports failed sends. Pass a logger without this core so a failing
// alert cannot raise another alert.
func NewCore(notifier Notifier, cfg Config, log *zap.Logger) *Core {
	messages := make(map[string]struct{}, len(cfg.Messages))
	for _, m := range cfg.Messages {
		messages[m] = struct{}{}
	}
	return &Core{dispatcher: &dispatcher{
		notifier: notifier,
		messages: messages,
		interval: cfg.Interval,
		timeout:  cfg.Timeout,
		log:      log,
	}}
}

func (c *Core) Enabled(level zapcore.Level) bool { return level >= zapcore.ErrorLevel }

// Keeps fields such as request_id that the request logger added with With.
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	return &Core{fields: append(slices.Clone(c.fields), fields...), dispatcher: c.dispatcher}
}

func (c *Core) Check(ent zapcore.Entry, //nolint:gocritic // signature fixed by zapcore.Core
	ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) && c.dispatcher.watches(ent.Message) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *Core) Write(ent zapcore.Entry, //nolint:gocritic // signature fixed by zapcore.Core
	fields []zapcore.Field) error {
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range c.fields {
		f.AddTo(enc)
	}
	for _, f := range fields {
		f.AddTo(enc)
	}

	e := ServerError{
		Time:      ent.Time,
		Message:   ent.Message,
		RequestID: stringField(enc.Fields, requestIDField),
		Error:     stringField(enc.Fields, errorField),
		Stack:     ent.Stack,
	}
	if ent.Caller.Defined {
		e.Caller = ent.Caller.TrimmedPath()
	}
	c.dispatcher.dispatch(&e)
	return nil
}

func (c *Core) Sync() error { return nil }

func stringField(fields map[string]any, key string) string {
	value, ok := fields[key].(string)
	if !ok {
		return ""
	}
	return value
}

type dispatcher struct {
	notifier Notifier
	messages map[string]struct{}
	interval time.Duration
	timeout  time.Duration
	log      *zap.Logger

	mu         sync.Mutex
	lastSent   time.Time
	suppressed int
}

func (d *dispatcher) watches(message string) bool {
	_, ok := d.messages[message]
	return ok
}

func (d *dispatcher) dispatch(e *ServerError) {
	if !d.claim(e) {
		return
	}
	go d.send(e)
}

// A burst of failures would otherwise send one email each and use up the
// mail provider's quota; the next alert reports how many were skipped.
func (d *dispatcher) claim(e *ServerError) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.lastSent.IsZero() && e.Time.Sub(d.lastSent) < d.interval {
		d.suppressed++
		return false
	}
	d.lastSent = e.Time
	e.Suppressed = d.suppressed
	d.suppressed = 0
	return true
}

// Runs in its own goroutine with a fresh context: a log call has no request
// context, and a slow mail API must not delay the failing response.
func (d *dispatcher) send(e *ServerError) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	if err := d.notifier.NotifyServerError(ctx, e); err != nil {
		d.log.Warn("server error alert not sent", zap.Error(err))
	}
}
