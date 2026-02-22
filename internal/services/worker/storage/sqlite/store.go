package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sqlitemigrate "github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/worker/storage"
	"github.com/louisbranch/fracturing.space/internal/services/worker/storage/sqlite/migrations"
	_ "modernc.org/sqlite"
)

// Store provides SQLite-backed worker attempt persistence.
type Store struct {
	sqlDB *sql.DB
}

// Open opens a worker SQLite store and applies migrations.
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

	store := &Store{sqlDB: sqlDB}
	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrations.FS, ""); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return store, nil
}

// Close releases the SQLite connection.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// RecordAttempt persists one worker processing attempt.
func (s *Store) RecordAttempt(ctx context.Context, attempt storage.AttemptRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}

	attempt.EventID = strings.TrimSpace(attempt.EventID)
	attempt.EventType = strings.TrimSpace(attempt.EventType)
	attempt.Consumer = strings.TrimSpace(attempt.Consumer)
	attempt.Outcome = strings.TrimSpace(attempt.Outcome)
	attempt.LastError = strings.TrimSpace(attempt.LastError)
	if attempt.EventID == "" {
		return fmt.Errorf("event id is required")
	}
	if attempt.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	if attempt.Consumer == "" {
		return fmt.Errorf("consumer is required")
	}
	if attempt.Outcome == "" {
		return fmt.Errorf("outcome is required")
	}
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}

	_, err := s.sqlDB.ExecContext(ctx, `
INSERT INTO worker_attempts (
	event_id,
	event_type,
	consumer,
	outcome,
	attempt_count,
	last_error,
	created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
`,
		attempt.EventID,
		attempt.EventType,
		attempt.Consumer,
		attempt.Outcome,
		attempt.AttemptCount,
		attempt.LastError,
		attempt.CreatedAt.UTC().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("record attempt: %w", err)
	}
	return nil
}

// ListAttempts lists newest-first attempt records.
func (s *Store) ListAttempts(ctx context.Context, limit int) ([]storage.AttemptRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.sqlDB.QueryContext(ctx, `
SELECT
	id,
	event_id,
	event_type,
	consumer,
	outcome,
	attempt_count,
	last_error,
	created_at
FROM worker_attempts
ORDER BY created_at DESC, id DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("list attempts: %w", err)
	}
	defer rows.Close()

	records := make([]storage.AttemptRecord, 0, limit)
	for rows.Next() {
		var record storage.AttemptRecord
		var createdAt int64
		if err := rows.Scan(
			&record.ID,
			&record.EventID,
			&record.EventType,
			&record.Consumer,
			&record.Outcome,
			&record.AttemptCount,
			&record.LastError,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		record.CreatedAt = time.UnixMilli(createdAt).UTC()
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attempts: %w", err)
	}
	return records, nil
}

var _ storage.AttemptStore = (*Store)(nil)
