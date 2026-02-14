package migrations

import (
	"io/fs"
	"sort"
	"testing"
)

func TestMigrationsEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected migrations to be embedded")
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if files[0] != "001_user_sessions.sql" {
		t.Fatalf("expected first migration 001_user_sessions.sql, got %s", files[0])
	}
}
