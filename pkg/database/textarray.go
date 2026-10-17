package database

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// TextArray maps a PostgreSQL text[] column. database/sql cannot scan arrays
// on its own, so scanning goes through pgx's type map.
type TextArray []string

func (a *TextArray) Scan(src any) error {
	if src == nil {
		*a = TextArray{}
		return nil
	}
	var out []string
	if err := pgtype.NewMap().SQLScanner(&out).Scan(src); err != nil {
		return fmt.Errorf("scan text array: %w", err)
	}
	*a = out
	return nil
}

// Value writes an array literal so the value works with any driver and with
// pgx's simple protocol. Every element is quoted, which also covers NULL and
// empty strings.
func (a TextArray) Value() (driver.Value, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, s := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(arrayElementEscaper.Replace(s))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String(), nil
}

var arrayElementEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)
