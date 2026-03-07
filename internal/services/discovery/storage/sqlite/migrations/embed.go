package migrations

import "embed"

// FS contains embedded SQLite migrations for discovery storage.
//
//go:embed *.sql
var FS embed.FS
