package migrations

import (
	"io/fs"
	"sort"
	"testing"
)

func TestEventsMigrationsEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(EventsFS, "events")
	if err != nil {
		t.Fatalf("read events migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected events migrations to be embedded")
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if files[0] != "001_events.sql" {
		t.Fatalf("expected first events migration 001_events.sql, got %s", files[0])
	}
}

func TestProjectionMigrationsEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(ProjectionsFS, "projections")
	if err != nil {
		t.Fatalf("read projection migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected projection migrations to be embedded")
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if files[0] != "001_projections.sql" {
		t.Fatalf("expected first projection migration 001_projections.sql, got %s", files[0])
	}
}
