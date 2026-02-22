package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenStoreCreatesDirectoryAndOpensDB(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "nested", "admin.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if store == nil {
		t.Fatal("expected store")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOpenStoreReturnsWrappedErrorWhenDirectoryCreationFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	blockingFile := filepath.Join(root, "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	dbPath := filepath.Join(blockingFile, "nested", "admin.db")

	_, err := OpenStore(dbPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create storage dir") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenStoreReturnsWrappedErrorWhenSQLiteOpenFails(t *testing.T) {
	t.Parallel()

	// Pointing at a directory path forces sqlite open to fail deterministically.
	dbPath := t.TempDir()

	_, err := OpenStore(dbPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "open admin sqlite store") {
		t.Fatalf("unexpected error: %v", err)
	}
}
