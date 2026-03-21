// Package migrations embeds SQLite schema migrations for the invite service.
package migrations

import "embed"

// FS contains embedded SQLite migrations for invite storage.
//
//go:embed *.sql
var FS embed.FS
