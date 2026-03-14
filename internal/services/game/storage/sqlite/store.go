package sqlite

import (
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
)

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

// fromMillis reverses toMillis for persisted millisecond timestamps.
func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// toNullMillis maps optional domain times to sql.NullInt64 for nullable DB columns.
func toNullMillis(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: toMillis(*value), Valid: true}
}

// fromNullMillis maps nullable SQL timestamps back into optional domain time values.
func fromNullMillis(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := fromMillis(value.Int64)
	return &t
}

// Store provides a SQLite-backed store implementing all storage interfaces.
type Store struct {
	sqlDB                        *sql.DB
	q                            *db.Queries
	tx                           *sql.Tx
	keyring                      *integrity.Keyring
	eventRegistry                *event.Registry
	projectionApplyOutboxEnabled bool
}

// txStore returns a shallow clone of the Store that routes all queries through
// the given transaction. The original Store is not mutated. This is used by
// ApplyProjectionEventExactlyOnce so the caller's apply callback operates
// inside the same transaction that reserves the idempotency checkpoint.
func (s *Store) txStore(tx *sql.Tx) *Store {
	if s == nil || tx == nil {
		return s
	}
	cloned := *s
	cloned.tx = tx
	cloned.q = s.q.WithTx(tx)
	return &cloned
}

// OpenEventsOption configures event-store behavior.
type OpenEventsOption func(*Store)

// WithProjectionApplyOutboxEnabled toggles enqueueing projection-apply work for appended events.
func WithProjectionApplyOutboxEnabled(enabled bool) OpenEventsOption {
	return func(s *Store) {
		s.projectionApplyOutboxEnabled = enabled
	}
}

// Open opens a SQLite projections store at the provided path.
//
// This is the historic convenience constructor used by some startup codepaths that
// only need projection storage.
func Open(path string) (*Store, error) {
	return OpenProjections(path)
}

// OpenEvents opens a SQLite event journal store at the provided path.
//
// This path wires integrity key material and the event registry so every appended
// event can be consistently hashed and validated in one place.
func OpenEvents(path string, keyring *integrity.Keyring, registry *event.Registry, opts ...OpenEventsOption) (*Store, error) {
	store, err := openStore(path, migrations.EventsFS, "events", keyring)
	if err != nil {
		return nil, err
	}
	store.eventRegistry = registry
	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}
	return store, nil
}

// OpenProjections opens a SQLite projections store at the provided path.
func OpenProjections(path string) (*Store, error) {
	return openStore(path, migrations.ProjectionsFS, "projections", nil)
}

// OpenContent opens a SQLite content catalog store at the provided path.
func OpenContent(path string) (*Store, error) {
	return openStore(path, migrations.ContentFS, "content", nil)
}

// Close closes the underlying SQLite database.
//
// Close is intentionally nil-safe so callers can defer it in all startup paths.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// openStore boots a SQLite bundle for a domain purpose (events/projections/content)
// and applies embedded migrations before the store is handed to higher layers.
func openStore(path string, migrationFS fs.FS, migrationRoot string, keyring *integrity.Keyring) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}

	store := &Store{
		sqlDB:   sqlDB,
		q:       db.New(sqlDB),
		keyring: keyring,
	}

	if err := runMigrations(sqlDB, migrationFS, migrationRoot); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// runMigrations executes embedded SQL migrations from the provided migration set.
// Files are sorted lexicographically to make startup behavior deterministic.
func runMigrations(sqlDB *sql.DB, migrationFS fs.FS, migrationRoot string) error {
	return sqlitemigrate.ApplyMigrations(sqlDB, migrationFS, migrationRoot)
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
