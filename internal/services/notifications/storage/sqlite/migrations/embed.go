package migrations

import "embed"

// FS contains embedded notifications SQLite migrations.
//
//go:embed *.sql
var FS embed.FS
