package migrations

import "embed"

// FS contains embedded SQLite migrations for connections storage.
//
//go:embed *.sql
var FS embed.FS
