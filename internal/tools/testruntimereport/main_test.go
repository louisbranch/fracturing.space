package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSummaryFromInputDir(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(
		t,
		filepath.Join(tempDir, "unit.jsonl"),
		`{"Action":"pass","Package":"example/pkg","Test":"TestSlow","Elapsed":0.400}`+"\n"+
			`{"Action":"pass","Package":"example/pkg","Elapsed":1.250}`+"\n",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"-input-dir", tempDir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "- unit: status=pass elapsed=1.250s package=example/pkg") {
		t.Fatalf("expected run summary output, got %q", stdout.String())
	}
}

func TestRunBudgetEnforcement(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(
		t,
		filepath.Join(tempDir, "unit.jsonl"),
		`{"Action":"pass","Package":"example/pkg","Elapsed":1.250}`+"\n",
	)
	budgetPath := filepath.Join(tempDir, "budget.json")
	writeFile(
		t,
		budgetPath,
		`{"runs":{"unit":{"baseline_seconds":1.0,"allowed_regression_pct":0}}}`+"\n",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{
		"-input-dir", tempDir,
		"-budget-file", budgetPath,
		"-enforce-budget",
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if code := exitCode(err); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "RUNTIME_BUDGET_WARNING: unit elapsed 1.250s exceeds threshold 1.000s") {
		t.Fatalf("expected runtime budget warning, got %q", stderr.String())
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
