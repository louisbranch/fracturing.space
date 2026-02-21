package sqlitemigrate

import (
	"database/sql"
	"testing"

	"testing/fstest"

	_ "modernc.org/sqlite"
)

func TestApplyMigrationsRecordsApplied(t *testing.T) {
	db := openInMemoryDB(t)

	migrations := fstest.MapFS{
		"001_create.sql": &fstest.MapFile{
			Data: []byte("-- +migrate Up\nCREATE TABLE items(id TEXT PRIMARY KEY);"),
		},
	}

	if err := ApplyMigrations(db, migrations, ""); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	rows := queryInt64(t, db, "SELECT COUNT(*) FROM schema_migrations")
	if rows != 1 {
		t.Fatalf("expected 1 migration row, got %d", rows)
	}

	if !tableExists(t, db, "items") {
		t.Fatal("expected applied table to exist")
	}
}

func TestApplyMigrationsSkipsAlreadyApplied(t *testing.T) {
	db := openInMemoryDB(t)

	first := fstest.MapFS{
		"001_create.sql": &fstest.MapFile{
			Data: []byte("-- +migrate Up\nCREATE TABLE items(id TEXT PRIMARY KEY);"),
		},
	}
	if err := ApplyMigrations(db, first, ""); err != nil {
		t.Fatalf("apply initial migrations: %v", err)
	}

	second := fstest.MapFS{
		"001_create.sql": &fstest.MapFile{
			Data: []byte("-- +migrate Up\nCREATE TABLE items(id TEXT PRIMARY KEY);"),
		},
	}
	if err := ApplyMigrations(db, second, ""); err != nil {
		t.Fatalf("re-apply migrations should be idempotent: %v", err)
	}

	rows := queryInt64(t, db, "SELECT COUNT(*) FROM schema_migrations")
	if rows != 1 {
		t.Fatalf("expected single migration row after replay, got %d", rows)
	}
}

func TestApplyMigrationsDoesNotRecordFailedMigration(t *testing.T) {
	db := openInMemoryDB(t)

	bad := fstest.MapFS{
		"001_bad.sql": &fstest.MapFile{
			Data: []byte("-- +migrate Up\nCREAT table things(id INT);"),
		},
	}
	if err := ApplyMigrations(db, bad, ""); err == nil {
		t.Fatalf("expected bad migration to fail")
	}

	rows := queryInt64(t, db, "SELECT COUNT(*) FROM schema_migrations")
	if rows != 0 {
		t.Fatalf("expected failed migration to stay unrecorded, got %d rows", rows)
	}

	good := fstest.MapFS{
		"001_bad.sql": &fstest.MapFile{
			Data: []byte("-- +migrate Up\nCREATE TABLE things(id INTEGER PRIMARY KEY);"),
		},
	}
	if err := ApplyMigrations(db, good, ""); err != nil {
		t.Fatalf("apply fixed migration: %v", err)
	}

	rows = queryInt64(t, db, "SELECT COUNT(*) FROM schema_migrations")
	if rows != 1 {
		t.Fatalf("expected fixed migration to be recorded, got %d rows", rows)
	}
}

func TestApplyMigrationsRespectsMigrationRoot(t *testing.T) {
	db := openInMemoryDB(t)

	migrations := fstest.MapFS{
		"events/001_events.sql": &fstest.MapFile{
			Data: []byte("-- +migrate Up\nCREATE TABLE event_rows(id TEXT PRIMARY KEY);"),
		},
	}

	if err := ApplyMigrations(db, migrations, "events"); err != nil {
		t.Fatalf("apply migrations with root: %v", err)
	}

	key := queryString(t, db, "SELECT name FROM schema_migrations LIMIT 1")
	if key != "events/001_events.sql" {
		t.Fatalf("expected migration key with root path, got %q", key)
	}

	if !tableExists(t, db, "event_rows") {
		t.Fatal("expected migrated table in root-based migration")
	}
}

func openInMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})
	return db
}

func queryInt64(t *testing.T, db *sql.DB, query string) int64 {
	t.Helper()
	var value int64
	row := db.QueryRow(query)
	if err := row.Scan(&value); err != nil {
		t.Fatalf("query int value: %v", err)
	}
	return value
}

func queryString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var value string
	row := db.QueryRow(query)
	if err := row.Scan(&value); err != nil {
		t.Fatalf("query string value: %v", err)
	}
	return value
}

func tableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name = ?"
	var name string
	row := db.QueryRow(query, tableName)
	if err := row.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		t.Fatalf("check table exists: %v", err)
	}
	return name == tableName
}
