package database

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// QueryTimer adds up the time a request spends waiting on the database, so
// the API can report its own time separately (see middleware.ServerTiming).
// Safe for concurrent queries within one request.
type QueryTimer struct {
	nanos atomic.Int64
	count atomic.Int64
}

// Total is the summed query time and the number of queries.
func (t *QueryTimer) Total() (time.Duration, int64) {
	return time.Duration(t.nanos.Load()), t.count.Load()
}

type queryTimerKey struct{}

// WithQueryTimer returns a context whose GORM queries are timed into the
// returned timer.
func WithQueryTimer(ctx context.Context) (context.Context, *QueryTimer) {
	t := &QueryTimer{}
	return context.WithValue(ctx, queryTimerKey{}, t), t
}

func queryTimerFrom(ctx context.Context) *QueryTimer {
	if ctx == nil {
		return nil
	}
	t, _ := ctx.Value(queryTimerKey{}).(*QueryTimer) //nolint:errcheck // absent means "not timed"
	return t
}

const queryStartKey = "fluentfox:query_start"

// registerQueryTiming hooks every GORM operation. The cost is two map writes
// and a clock read per query; contexts without a timer are skipped.
func registerQueryTiming(db *gorm.DB) error {
	before := func(tx *gorm.DB) {
		if queryTimerFrom(tx.Statement.Context) != nil {
			tx.InstanceSet(queryStartKey, time.Now())
		}
	}
	after := func(tx *gorm.DB) {
		t := queryTimerFrom(tx.Statement.Context)
		if t == nil {
			return
		}
		if start, ok := tx.InstanceGet(queryStartKey); ok {
			if at, ok := start.(time.Time); ok {
				t.nanos.Add(int64(time.Since(at)))
				t.count.Add(1)
			}
		}
	}

	cb := db.Callback()
	err := errors.Join(
		cb.Create().Before("gorm:create").Register("fluentfox:timing_before_create", before),
		cb.Create().After("gorm:create").Register("fluentfox:timing_after_create", after),
		cb.Query().Before("gorm:query").Register("fluentfox:timing_before_query", before),
		cb.Query().After("gorm:query").Register("fluentfox:timing_after_query", after),
		cb.Update().Before("gorm:update").Register("fluentfox:timing_before_update", before),
		cb.Update().After("gorm:update").Register("fluentfox:timing_after_update", after),
		cb.Delete().Before("gorm:delete").Register("fluentfox:timing_before_delete", before),
		cb.Delete().After("gorm:delete").Register("fluentfox:timing_after_delete", after),
		cb.Row().Before("gorm:row").Register("fluentfox:timing_before_row", before),
		cb.Row().After("gorm:row").Register("fluentfox:timing_after_row", after),
		cb.Raw().Before("gorm:raw").Register("fluentfox:timing_before_raw", before),
		cb.Raw().After("gorm:raw").Register("fluentfox:timing_after_raw", after),
	)
	if err != nil {
		return fmt.Errorf("database: register query timing: %w", err)
	}
	return nil
}
