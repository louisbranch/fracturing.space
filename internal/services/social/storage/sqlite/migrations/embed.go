package migrations

import "embed"

// FS contains embedded SQLite migrations for social storage.
//
//go:embed *.sql
var FS embed.FS
