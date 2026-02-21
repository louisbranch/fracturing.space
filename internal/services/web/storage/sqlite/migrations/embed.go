package migrations

import "embed"

// FS provides access to embedded cache migrations.
//
//go:embed *.sql
var FS embed.FS
