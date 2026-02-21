package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/admin/storage"
	"github.com/louisbranch/fracturing.space/internal/services/admin/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/admin/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

const timeFormat = time.RFC3339Nano

// Store provides a SQLite-backed store implementing admin storage interfaces.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

// Open opens a SQLite store at the provided path.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(path)
	dsn := cleanPath + "?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	store := &Store{
		sqlDB: sqlDB,
		q:     db.New(sqlDB),
	}

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

// runMigrations runs embedded SQL migrations.
func (s *Store) runMigrations() error {
	return sqlitemigrate.ApplyMigrations(s.sqlDB, migrations.FS, "")
}

// extractUpMigration extracts the Up migration portion from a migration file.
func extractUpMigration(content string) string {
	return sqlitemigrate.ExtractUpMigration(content)
}

// isAlreadyExistsError checks if the error is a table/index already exists error.
func isAlreadyExistsError(err error) bool {
	return sqlitemigrate.IsAlreadyExistsError(err)
}

// PutUserSession persists a user session record.
func (s *Store) PutUserSession(ctx context.Context, sessionID string, createdAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return s.q.PutUserSession(ctx, db.PutUserSessionParams{
		SessionID: sessionID,
		CreatedAt: createdAt.Format(timeFormat),
	})
}

var _ storage.Store = (*Store)(nil)
