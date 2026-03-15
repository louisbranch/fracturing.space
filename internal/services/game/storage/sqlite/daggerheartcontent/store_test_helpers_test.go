package daggerheartcontent

import (
	"path/filepath"
	"testing"
)

func openTestContentStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "content.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open content store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close content store: %v", err)
		}
	})
	return store
}
