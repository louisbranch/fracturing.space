package sqliteconn

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenRequiresPath(t *testing.T) {
	t.Parallel()

	if _, err := Open(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestOpenAppliesExpectedPragmas(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "store.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	})

	if err := verifyPragmas(db); err != nil {
		t.Fatalf("verify pragmas: %v", err)
	}
}

func TestVerifyPragmasDetectsMismatch(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "default.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	})
	if err := db.Ping(); err != nil {
		t.Fatalf("ping sqlite: %v", err)
	}

	if err := verifyPragmas(db); err == nil {
		t.Fatal("expected pragma verification error")
	}
}

func TestIsBusyOrLockedError(t *testing.T) {
	t.Parallel()

	busyErr := generateBusyError(t)
	if !IsBusyOrLockedError(busyErr) {
		t.Fatalf("expected busy/locked error, got %v", busyErr)
	}
	if IsBusyOrLockedError(sql.ErrNoRows) {
		t.Fatal("expected false for non-sqlite busy error")
	}
}

func generateBusyError(t *testing.T) error {
	t.Helper()

	path := filepath.Join(t.TempDir(), "busy.db")
	dsn := path + "?_pragma=busy_timeout(0)"

	db1, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}
	defer db1.Close()

	db2, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()

	if _, err := db1.Exec("CREATE TABLE locks (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	tx, err := db1.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("INSERT INTO locks (id) VALUES (1)"); err != nil {
		t.Fatalf("insert in tx: %v", err)
	}

	_, busyErr := db2.Exec("INSERT INTO locks (id) VALUES (2)")
	if busyErr == nil {
		t.Fatal("expected busy/locked error")
	}

	return busyErr
}
