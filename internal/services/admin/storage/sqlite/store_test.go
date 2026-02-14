package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestPutUserSessionStoresTimestamp(t *testing.T) {
	store := openTempStore(t)

	createdAt := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	if err := store.PutUserSession(context.Background(), "session-1", createdAt); err != nil {
		t.Fatalf("put user session: %v", err)
	}

	var storedID string
	var storedAt string
	row := store.sqlDB.QueryRow("SELECT session_id, created_at FROM user_sessions WHERE session_id = ?", "session-1")
	if err := row.Scan(&storedID, &storedAt); err != nil {
		t.Fatalf("scan user session: %v", err)
	}
	if storedID != "session-1" {
		t.Fatalf("expected session id session-1, got %s", storedID)
	}
	if storedAt != createdAt.Format(timeFormat) {
		t.Fatalf("expected created_at %s, got %s", createdAt.Format(timeFormat), storedAt)
	}
}

func TestPutUserSessionDefaultsTime(t *testing.T) {
	store := openTempStore(t)

	if err := store.PutUserSession(context.Background(), "session-2", time.Time{}); err != nil {
		t.Fatalf("put user session: %v", err)
	}

	var storedAt string
	row := store.sqlDB.QueryRow("SELECT created_at FROM user_sessions WHERE session_id = ?", "session-2")
	if err := row.Scan(&storedAt); err != nil {
		t.Fatalf("scan user session: %v", err)
	}
	if storedAt == "" {
		t.Fatal("expected created_at to be set")
	}
}

func TestPutUserSessionValidation(t *testing.T) {
	store := openTempStore(t)

	if err := store.PutUserSession(context.Background(), "", time.Now()); err == nil {
		t.Fatal("expected error for empty session id")
	}
}

func TestPutUserSessionRequiresStore(t *testing.T) {
	var store *Store
	if err := store.PutUserSession(context.Background(), "session-3", time.Now()); err == nil {
		t.Fatal("expected error for nil store")
	}
}

func openTempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "admin.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil && err != sql.ErrConnDone {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}
