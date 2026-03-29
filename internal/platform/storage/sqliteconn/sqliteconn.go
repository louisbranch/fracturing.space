package sqliteconn

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	busyTimeoutMillis   = 5000
	synchronousMode     = 1 // PRAGMA synchronous=NORMAL
	expectedJournalMode = "wal"
)

// Open opens a SQLite database using the project-wide connection policy.
//
// It applies required pragmas at open time and verifies they took effect before
// returning the handle.
func Open(path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(path)
	dsn := fmt.Sprintf(
		"%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(%d)&_pragma=synchronous(NORMAL)",
		cleanPath,
		busyTimeoutMillis,
	)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}
	if err := verifyPragmas(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

// IsBusyOrLockedError reports whether err wraps SQLITE_BUSY or SQLITE_LOCKED.
func IsBusyOrLockedError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	switch sqliteErr.Code() & 0xff {
	case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED:
		return true
	default:
		return false
	}
}

func verifyPragmas(sqlDB *sql.DB) error {
	if sqlDB == nil {
		return fmt.Errorf("sqlite db is required")
	}

	var journalMode string
	if err := sqlDB.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("check sqlite journal_mode pragma: %w", err)
	}
	if !strings.EqualFold(journalMode, expectedJournalMode) {
		return fmt.Errorf("sqlite journal_mode mismatch: got %q want %q", journalMode, expectedJournalMode)
	}

	var foreignKeys int
	if err := sqlDB.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return fmt.Errorf("check sqlite foreign_keys pragma: %w", err)
	}
	if foreignKeys != 1 {
		return fmt.Errorf("sqlite foreign_keys mismatch: got %d want 1", foreignKeys)
	}

	var busyTimeout int
	if err := sqlDB.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		return fmt.Errorf("check sqlite busy_timeout pragma: %w", err)
	}
	if busyTimeout != busyTimeoutMillis {
		return fmt.Errorf("sqlite busy_timeout mismatch: got %d want %d", busyTimeout, busyTimeoutMillis)
	}

	var synchronous int
	if err := sqlDB.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		return fmt.Errorf("check sqlite synchronous pragma: %w", err)
	}
	if synchronous != synchronousMode {
		return fmt.Errorf("sqlite synchronous mismatch: got %d want %d", synchronous, synchronousMode)
	}

	return nil
}
