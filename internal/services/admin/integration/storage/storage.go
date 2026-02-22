package storage

import (
	"fmt"
	"os"
	"path/filepath"

	adminsqlite "github.com/louisbranch/fracturing.space/internal/services/admin/storage/sqlite"
)

// OpenStore opens the admin SQLite store and creates its parent directory when needed.
func OpenStore(path string) (*adminsqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}

	store, err := adminsqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open admin sqlite store: %w", err)
	}
	return store, nil
}
