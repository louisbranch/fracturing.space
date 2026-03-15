package daggerheartcontent

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
)

// Store provides the SQLite-backed Daggerheart catalog backend.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

var (
	_ contentstore.DaggerheartContentStore          = (*Store)(nil)
	_ contentstore.DaggerheartCatalogReadinessStore = (*Store)(nil)
)

// Open opens a SQLite content catalog store at the provided path.
func Open(path string) (*Store, error) {
	return openStore(path)
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

func openStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		return nil, err
	}

	store := &Store{
		sqlDB: sqlDB,
		q:     db.New(sqlDB),
	}

	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.ContentFS, "content", time.Now); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}
