package database

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// Float64Array maps a nullable PostgreSQL double precision[] column. A nil
// slice is stored as NULL, unlike TextArray, because NULL has its own meaning
// in the columns that use it.
type Float64Array []float64

func (a *Float64Array) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var out []float64
	if err := pgtype.NewMap().SQLScanner(&out).Scan(src); err != nil {
		return fmt.Errorf("scan float64 array: %w", err)
	}
	*a = out
	return nil
}

func (a Float64Array) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	parts := make([]string, len(a))
	for i, f := range a {
		parts[i] = strconv.FormatFloat(f, 'g', -1, 64)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}
