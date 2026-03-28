package eventjournal

import (
	"database/sql"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sqliteintegrationoutbox "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/integrationoutbox"
)

// DB exposes the underlying SQLite handle for sibling backend packages that
// bind to the same migrated database file.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.sqlDB
}

// IntegrationOutboxStore binds the integration-outbox backend to this event
// store's SQLite database so worker-facing delivery persistence is explicit.
func (s *Store) IntegrationOutboxStore() storage.IntegrationOutboxWorkerStore {
	if s == nil {
		return nil
	}
	return sqliteintegrationoutbox.Bind(s.sqlDB)
}
