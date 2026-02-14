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

	store, err := openProjectionStore(path)
	if err != nil {
		t.Fatalf("open projection store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close projection store: %v", err)
	}
}

func TestOpenEventStoreRequiresKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")

	if _, err := openEventStore(path); err == nil {
		t.Fatal("expected error when HMAC key is missing")
	}
}

func TestOpenEventStoreSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	store, err := openEventStore(path)
	if err != nil {
		t.Fatalf("open event store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close event store: %v", err)
	}
}

func TestOpenStorageBundleSuccess(t *testing.T) {
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	srvEnv := serverEnv{
		EventsDBPath:      filepath.Join(base, "events.db"),
		ProjectionsDBPath: filepath.Join(base, "projections.db"),
		ContentDBPath:     filepath.Join(base, "content.db"),
	}
	bundle, err := openStorageBundle(srvEnv)
	if err != nil {
		t.Fatalf("open storage bundle: %v", err)
	}
	bundle.Close()
}

func TestOpenStorageBundleProjectionFailure(t *testing.T) {
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")

	// Point projections at a file (not a directory) to force failure.
	blocker := filepath.Join(base, "blocker")
	if err := os.WriteFile(blocker, []byte("data"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	srvEnv := serverEnv{
		EventsDBPath:      filepath.Join(base, "events.db"),
		ProjectionsDBPath: filepath.Join(blocker, "projections.db"),
		ContentDBPath:     filepath.Join(base, "content.db"),
	}
	if _, err := openStorageBundle(srvEnv); err == nil {
		t.Fatal("expected error when projection store fails to open")
	}
}

func TestDialAuthGRPCTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if _, _, err := dialAuthGRPC(ctx, "127.0.0.1:1"); err == nil {
		t.Fatal("expected dial auth error")
	}
}
