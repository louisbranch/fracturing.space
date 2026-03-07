package migrations

import "embed"

// FS contains embedded SQLite migrations for status override storage.
//
//go:embed *.sql
var FS embed.FS
