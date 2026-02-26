package server

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	socialsqlite "github.com/louisbranch/fracturing.space/internal/services/social/storage/sqlite"
	_ "modernc.org/sqlite"
)

func TestNewWithAddr_DoesNotMigrateAuthContacts(t *testing.T) {
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

	socialPath := filepath.Join(t.TempDir(), "social.db")
	t.Setenv("FRACTURING_SPACE_SOCIAL_DB_PATH", socialPath)
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", authPath)
	t.Setenv("FRACTURING_SPACE_SOCIAL_MIGRATE_AUTH_CONTACTS", "true")

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	srv.Close()

	socialStore, err := socialsqlite.Open(socialPath)
	if err != nil {
		t.Fatalf("open social store: %v", err)
	}
	t.Cleanup(func() { _ = socialStore.Close() })

	contacts, err := socialStore.ListContacts(t.Context(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(contacts.Contacts) != 0 {
		t.Fatalf("contacts len = %d, want 0", len(contacts.Contacts))
	}
}
