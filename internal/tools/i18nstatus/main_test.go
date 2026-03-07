package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesOutputs(t *testing.T) {
	tempDir := t.TempDir()
	markdownPath := filepath.Join(tempDir, "i18n-status.md")
	jsonPath := filepath.Join(tempDir, "i18n-status.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{
		"-out", markdownPath,
		"-json-out", jsonPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "wrote "+markdownPath+" and "+jsonPath) {
		t.Fatalf("expected success output, got %q", stdout.String())
	}

	markdownData, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	if !strings.Contains(string(markdownData), "# I18n Status") {
		t.Fatalf("expected markdown header, got %q", string(markdownData))
	}

	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json output: %v", err)
	}
	if !strings.Contains(string(jsonData), `"base_locale"`) {
		t.Fatalf("expected json report body, got %q", string(jsonData))
	}
}

func TestRunMissingBaseLocale(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"-base-locale", "zz-ZZ"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if code := exitCode(err); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `base locale "zz-ZZ" is missing from catalogs`) {
		t.Fatalf("expected missing locale error, got %q", stderr.String())
	}
}
