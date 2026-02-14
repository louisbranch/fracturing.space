package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureDirCreatesParent(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "nested", "store.db")

	if err := ensureDir(path); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
}

func TestEnsureDirRejectsFileParent(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	path := filepath.Join(file, "store.db")
	if err := ensureDir(path); err == nil {
		t.Fatal("expected error when parent is a file")
	}
}

func TestOpenProjectionStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projections.db")
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH", path)

	store, err := openProjectionStore()
	if err != nil {
		t.Fatalf("open projection store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close projection store: %v", err)
	}
}

func TestOpenEventStoreRequiresKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", path)
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")

	if _, err := openEventStore(); err == nil {
		t.Fatal("expected error when HMAC key is missing")
	}
}

func TestOpenEventStoreSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", path)
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore()
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close event store: %v", err)
	}
}

func TestDialAuthGRPCTimeout(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", "127.0.0.1:1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if _, _, err := dialAuthGRPC(ctx); err == nil {
		t.Fatal("expected dial auth error")
	}
}
