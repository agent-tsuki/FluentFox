// Package migrations holds the goose SQL migrations, embedded so the migrate
// binary needs no files on disk.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
