package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "i18n catalog check passed") {
		t.Fatalf("expected success output, got %q", stdout.String())
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
