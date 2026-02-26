package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCallViolations_DetectsDisallowedCalls(t *testing.T) {
	src := `package example

func foo() {
	s.stores.Domain.Execute()
	bar()
}
`
	path := writeTempGoFile(t, src)

	violations, err := ScanCallViolations(path, func(callPath string) bool {
		return callPath == "s.stores.Domain.Execute"
	})
	if err != nil {
		t.Fatalf("ScanCallViolations: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	if violations[0] != 4 {
		t.Errorf("expected violation on line 4, got line %d", violations[0])
	}
}

func TestScanCallViolations_NoViolations(t *testing.T) {
	src := `package example

func foo() {
	bar()
	baz()
}
`
	path := writeTempGoFile(t, src)

	violations, err := ScanCallViolations(path, func(callPath string) bool {
		return callPath == "s.stores.Domain.Execute"
	})
	if err != nil {
		t.Fatalf("ScanCallViolations: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestScanLiteralViolations_DetectsForbiddenStrings(t *testing.T) {
	src := `package example

func foo() {
	x := "action.outcome_rejected"
	y := "safe_string"
}
`
	path := writeTempGoFile(t, src)

	violations, err := ScanLiteralViolations(path, []string{"action.outcome_rejected"})
	if err != nil {
		t.Fatalf("ScanLiteralViolations: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	if violations[0] != 4 {
		t.Errorf("expected violation on line 4, got line %d", violations[0])
	}
}

func TestScanHandlerDir_ReturnsNonTestGoFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"handler.go", "handler_test.go", "util.go", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	files, err := ScanHandlerDir(dir)
	if err != nil {
		t.Fatalf("ScanHandlerDir: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	// Files should be sorted.
	if files[0] != "handler.go" || files[1] != "util.go" {
		t.Errorf("unexpected files: %v", files)
	}
}

func writeTempGoFile(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
