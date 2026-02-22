package server

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	connectionssqlite "github.com/louisbranch/fracturing.space/internal/services/connections/storage/sqlite"
	_ "modernc.org/sqlite"
)

func TestMigrateContactsFromAuth_CopiesDirectedContacts(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.db")
	authDB, err := sql.Open("sqlite", authPath)
	if err != nil {
		t.Fatalf("open auth sqlite: %v", err)
	}
	t.Cleanup(func() { _ = authDB.Close() })

	if _, err := authDB.Exec(`
CREATE TABLE user_contacts (
	owner_user_id TEXT NOT NULL,
	contact_user_id TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (owner_user_id, contact_user_id)
);
`); err != nil {
		t.Fatalf("create user_contacts: %v", err)
	}
	now := time.Date(2026, time.February, 22, 14, 0, 0, 0, time.UTC).UnixMilli()
	if _, err := authDB.Exec(
		`INSERT INTO user_contacts (owner_user_id, contact_user_id, created_at, updated_at) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		"user-1", "user-2", now, now,
		"user-1", "user-3", now, now,
	); err != nil {
		t.Fatalf("seed user_contacts: %v", err)
	}

	connectionsStore, err := connectionssqlite.Open(filepath.Join(t.TempDir(), "connections.db"))
	if err != nil {
		t.Fatalf("open connections store: %v", err)
	}
	t.Cleanup(func() { _ = connectionsStore.Close() })

	if err := migrateContactsFromAuth(context.Background(), authPath, connectionsStore); err != nil {
		t.Fatalf("migrate contacts: %v", err)
	}

	contacts, err := connectionsStore.ListContacts(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(contacts.Contacts) != 2 {
		t.Fatalf("contacts len = %d, want 2", len(contacts.Contacts))
	}
}
