package sqlite

import (
	"io/fs"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	sqlitecoreprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/coreprojection"
	sqliteeventjournal "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/eventjournal"
)

// Store is the compatibility alias for the extracted shared projection backend.
// Prefer importing `storage/sqlite/coreprojection` directly in new production
// code.
type Store = sqlitecoreprojection.Store

// Open opens the extracted shared projection backend.
func Open(path string) (*Store, error) {
	return sqlitecoreprojection.Open(path)
}

// OpenProjections opens the extracted shared projection backend.
func OpenProjections(path string) (*Store, error) {
	return sqlitecoreprojection.Open(path)
}

// OpenEventsOption is the compatibility alias for the extracted event journal
// open options. Prefer importing `storage/sqlite/eventjournal` directly in new
// code.
type OpenEventsOption = sqliteeventjournal.OpenOption

// WithProjectionApplyOutboxEnabled configures event-journal outbox enqueueing.
func WithProjectionApplyOutboxEnabled(enabled bool) OpenEventsOption {
	return sqliteeventjournal.WithProjectionApplyOutboxEnabled(enabled)
}

// OpenEvents opens the extracted event journal backend.
func OpenEvents(path string, keyring *integrity.Keyring, registry *event.Registry, opts ...OpenEventsOption) (*sqliteeventjournal.Store, error) {
	return sqliteeventjournal.Open(path, keyring, registry, opts...)
}

// openStore preserves the historic root-package test seam while the
// compatibility layer remains in place.
func openStore(path string, _ fs.FS, _ string) (*Store, error) {
	return sqlitecoreprojection.Open(path)
}

// extractUpMigration extracts the Up migration portion from a migration file.
// Down sections are intentionally ignored during startup execution.
func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

// isAlreadyExistsError checks if the error is a table/index already exists error.
func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}
