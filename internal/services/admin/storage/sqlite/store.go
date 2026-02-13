package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content, err := fs.ReadFile(migrations.FS, file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		upSQL := extractUpMigration(string(content))
		if upSQL == "" {
			continue
		}

		if _, err := s.sqlDB.Exec(upSQL); err != nil {
			if !isAlreadyExistsError(err) {
				return fmt.Errorf("exec migration %s: %w", file, err)
			}
		}
	}

	return nil
}

// extractUpMigration extracts the Up migration portion from a migration file.
func extractUpMigration(content string) string {
	upIdx := strings.Index(content, "-- +migrate Up")
	if upIdx == -1 {
		return content
	}
	downIdx := strings.Index(content, "-- +migrate Down")
	if downIdx == -1 {
		return content[upIdx+len("-- +migrate Up"):]
	}
	return content[upIdx+len("-- +migrate Up") : downIdx]
}

// isAlreadyExistsError checks if the error is a table/index already exists error.
func isAlreadyExistsError(err error) bool {
	return strings.Contains(err.Error(), "already exists")
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
