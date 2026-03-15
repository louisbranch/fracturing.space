package eventjournal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
)

type OpenOption func(*Store)

// WithProjectionApplyOutboxEnabled toggles enqueueing projection-apply work
// items while events are appended.
func WithProjectionApplyOutboxEnabled(enabled bool) OpenOption {
	return func(s *Store) {
		s.projectionApplyOutboxEnabled = enabled
	}
}

// Store owns the event-journal-backed SQLite surfaces: event append/query,
// audit writes, and event-owned outbox providers.
type Store struct {
	sqlDB                        *sql.DB
	q                            *db.Queries
	keyring                      *integrity.Keyring
	eventRegistry                *event.Registry
	projectionApplyOutboxEnabled bool
}

// Open opens a SQLite event journal store at the provided path.
func Open(path string, keyring *integrity.Keyring, registry *event.Registry, opts ...OpenOption) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}

	store := &Store{
		sqlDB:         sqlDB,
		q:             db.New(sqlDB),
		keyring:       keyring,
		eventRegistry: registry,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}

	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.EventsFS, "events", time.Now); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// Close closes the underlying SQLite database handle.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}
