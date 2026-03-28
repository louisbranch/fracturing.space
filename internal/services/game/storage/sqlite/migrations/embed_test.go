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

	if len(files) != 2 || files[0] != "001_events.sql" || files[1] != "002_projection_apply_campaign_leases.sql" {
		t.Fatalf("events migrations = %v, want [001_events.sql 002_projection_apply_campaign_leases.sql]", files)
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

	if len(files) != 1 || files[0] != "001_projections.sql" {
		t.Fatalf("projection migrations = %v, want [001_projections.sql]", files)
	}
}

func TestContentMigrationsEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(ContentFS, "content")
	if err != nil {
		t.Fatalf("read content migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected content migrations to be embedded")
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if len(files) != 1 || files[0] != "001_content.sql" {
		t.Fatalf("content migrations = %v, want [001_content.sql]", files)
	}
}
