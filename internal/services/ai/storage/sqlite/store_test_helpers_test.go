package sqlite

import (
	"path/filepath"
	"testing"
	"time"
)

func ptrTime(value time.Time) *time.Time {
	return &value
}

func openTempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ai.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}
