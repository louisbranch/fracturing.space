package migrations

import "embed"

// FS contains embedded SQLite migrations for listing storage.
//
//go:embed *.sql
var FS embed.FS
