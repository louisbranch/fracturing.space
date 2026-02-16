package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesCatalog(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	outPath := "docs/icon-catalog.md"
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"-root", root, "-out", outPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(root, outPath))
	if err != nil {
		t.Fatalf("read generated catalog: %v", err)
	}
	if !strings.Contains(string(data), "title: \"Icon Catalog\"") {
		t.Fatalf("catalog output missing title:\n%s", string(data))
	}
}

func TestRunResolvesRootFromWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}
	outPath := filepath.Join(root, "docs", "project", "icon-catalog.md")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated catalog: %v", err)
	}
	if !strings.Contains(string(data), "title: \"Icon Catalog\"") {
		t.Fatalf("catalog output missing title:\n%s", string(data))
	}
}

func TestRunReturnsErrorWhenRootMissing(t *testing.T) {
	root := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = run([]string{}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected root resolution error")
	}
	if !strings.Contains(err.Error(), "go.mod not found above") {
		t.Fatalf("error = %q, want go.mod not found", err.Error())
	}
}

func TestRunReturnsUsageErrorOnInvalidFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected parse error for invalid flag")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("error = %q, want invalid flag message", err.Error())
	}
}
