package migrations

import "embed"

// FS exposes play storage migrations.
//
//go:embed *.sql
var FS embed.FS
