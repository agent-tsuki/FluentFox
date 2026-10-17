// Package content holds the course content files. They are embedded so the
// migrate binary can load them on every deploy without the files on disk.
package content

import "embed"

//go:embed kana/*.json kanji/*.json
var FS embed.FS
