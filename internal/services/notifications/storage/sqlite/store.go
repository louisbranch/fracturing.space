package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage/sqlite/migrations"
)

// Store provides SQLite-backed persistence for notifications state.
type Store struct {
	sqlDB *sql.DB
}

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

// Open opens a notifications SQLite store at the provided path.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}

	store := &Store{sqlDB: sqlDB}
	if err := store.runMigrations(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return store, nil
}

// Close closes the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func (s *Store) runMigrations() error {
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "")
}

var _ storage.NotificationStore = (*Store)(nil)
var _ storage.DeliveryStore = (*Store)(nil)
var _ storage.NotificationBootstrapStore = (*Store)(nil)
