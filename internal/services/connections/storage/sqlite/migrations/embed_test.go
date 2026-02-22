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

	expected := []string{"001_connections.sql"}
	if len(files) != len(expected) {
		t.Fatalf("migration file count = %d, want %d (%v)", len(files), len(expected), expected)
	}
	for i := range expected {
		if files[i] != expected[i] {
			t.Fatalf("migration file[%d] = %q, want %q", i, files[i], expected[i])
		}
	}
}
