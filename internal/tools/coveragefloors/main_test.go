package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTryLoadFloorsOptionalFileStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, path string)
	}{
		{
			name: "missing file",
			setup: func(t *testing.T, path string) {
				t.Helper()
			},
		},
		{
			name: "empty file",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, nil, 0o644); err != nil {
					t.Fatalf("write empty file: %v", err)
				}
			},
		},
		{
			name: "whitespace-only file",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("  \n\t "), 0o644); err != nil {
					t.Fatalf("write whitespace file: %v", err)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "floors.json")
			tc.setup(t, path)

			got, err := tryLoadFloors(path)
			if err != nil {
				t.Fatalf("tryLoadFloors returned error: %v", err)
			}
			if got.Version != 0 {
				t.Fatalf("expected zero version, got %d", got.Version)
			}
			if got.AllowDrop != 0 {
				t.Fatalf("expected zero allow_drop, got %.1f", got.AllowDrop)
			}
			if len(got.Packages) != 0 {
				t.Fatalf("expected no packages, got %d", len(got.Packages))
			}
		})
	}
}

func TestTryLoadFloorsInvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "floors.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid floors file: %v", err)
	}

	if _, err := tryLoadFloors(path); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestRunCheckSkipsRemovedPackagesWhenSeedProvided(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "coverage.out")
	floorsPath := filepath.Join(tmpDir, "floors.json")
	seedPath := filepath.Join(tmpDir, "seed.json")

	// Profile only covers pkg/a — pkg/old was renamed/removed.
	profile := "mode: set\n" +
		"github.com/example/project/pkg/a/file.go:1.1,1.2 1 1\n"

	// Ratcheted floors (from badges) still reference the old package.
	floors := `{
  "version": 1,
  "allow_drop": 0.1,
  "packages": [
    {"package": "github.com/example/project/pkg/a", "floor": 90.0},
    {"package": "github.com/example/project/pkg/old", "floor": 95.0}
  ]
}`

	// Seed (local) only lists pkg/a — pkg/old was intentionally removed.
	seed := `{
  "version": 1,
  "allow_drop": 0.1,
  "packages": [
    {"package": "github.com/example/project/pkg/a", "floor": 90.0}
  ]
}`

	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	if err := os.WriteFile(floorsPath, []byte(floors), 0o644); err != nil {
		t.Fatalf("write floors: %v", err)
	}
	if err := os.WriteFile(seedPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	// Should pass — pkg/old is absent from seed, so it's skipped as "removed".
	if err := runCheck([]string{
		"-profile=" + profilePath,
		"-floors=" + floorsPath,
		"-seed=" + seedPath,
	}); err != nil {
		t.Fatalf("runCheck should pass when removed packages are absent from seed: %v", err)
	}
}

func TestRunCheckFailsMissingPackageWithoutSeed(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "coverage.out")
	floorsPath := filepath.Join(tmpDir, "floors.json")

	// Profile only covers pkg/a — pkg/old is missing.
	profile := "mode: set\n" +
		"github.com/example/project/pkg/a/file.go:1.1,1.2 1 1\n"

	floors := `{
  "version": 1,
  "allow_drop": 0.1,
  "packages": [
    {"package": "github.com/example/project/pkg/a", "floor": 90.0},
    {"package": "github.com/example/project/pkg/old", "floor": 95.0}
  ]
}`

	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	if err := os.WriteFile(floorsPath, []byte(floors), 0o644); err != nil {
		t.Fatalf("write floors: %v", err)
	}

	// Without seed, missing packages should fail.
	if err := runCheck([]string{
		"-profile=" + profilePath,
		"-floors=" + floorsPath,
	}); err == nil {
		t.Fatal("runCheck should fail when packages are missing without seed")
	}
}

func TestRunRatchetAllowsEmptyExistingFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "coverage.out")
	seedPath := filepath.Join(tmpDir, "seed.json")
	existingPath := filepath.Join(tmpDir, "existing.json")
	outPath := filepath.Join(tmpDir, "out.json")

	profile := "mode: set\n" +
		"github.com/example/project/pkg/file.go:1.1,1.2 1 1\n"
	seed := "{\n" +
		"  \"version\": 1,\n" +
		"  \"allow_drop\": 0.1,\n" +
		"  \"packages\": [\n" +
		"    {\n" +
		"      \"package\": \"github.com/example/project/pkg\",\n" +
		"      \"floor\": 10.0\n" +
		"    }\n" +
		"  ]\n" +
		"}\n"

	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	if err := os.WriteFile(seedPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	if err := os.WriteFile(existingPath, nil, 0o644); err != nil {
		t.Fatalf("write empty existing: %v", err)
	}

	if err := runRatchet([]string{
		"-profile=" + profilePath,
		"-seed=" + seedPath,
		"-existing=" + existingPath,
		"-out=" + outPath,
	}); err != nil {
		t.Fatalf("runRatchet returned error: %v", err)
	}

	floors, err := loadFloors(outPath)
	if err != nil {
		t.Fatalf("load output floors: %v", err)
	}
	if len(floors.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(floors.Packages))
	}
	if floors.Packages[0].Package != "github.com/example/project/pkg" {
		t.Fatalf("unexpected package: %s", floors.Packages[0].Package)
	}
	if floors.Packages[0].Floor != 100.0 {
		t.Fatalf("expected ratcheted floor 100.0, got %.1f", floors.Packages[0].Floor)
	}
}
