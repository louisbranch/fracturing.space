// Package migrations contains embedded SQL migrations for the SQLite store.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
