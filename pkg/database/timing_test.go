package database

import (
	"context"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type timedRow struct {
	ID   int
	Name string
}

// DryRun builds the SQL and runs every callback without a server, which is
// all the timing hooks need.
func dryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost dbname=none"}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registerQueryTiming(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestQueryTimingCountsQueriesInTimedContext(t *testing.T) {
	db := dryRunDB(t)
	ctx, timer := WithQueryTimer(context.Background())

	var rows []timedRow
	db.WithContext(ctx).Find(&rows)
	db.WithContext(ctx).Where("id = ?", 1).First(&timedRow{})
	db.WithContext(ctx).Create(&timedRow{Name: "a"})

	if _, n := timer.Total(); n != 3 {
		t.Fatalf("counted %d queries, want 3", n)
	}
}

func TestQueryTimingIgnoresUntimedContext(t *testing.T) {
	db := dryRunDB(t)
	_, timer := WithQueryTimer(context.Background())

	var rows []timedRow
	db.WithContext(context.Background()).Find(&rows)

	if _, n := timer.Total(); n != 0 {
		t.Fatalf("counted %d queries from another context, want 0", n)
	}
}
