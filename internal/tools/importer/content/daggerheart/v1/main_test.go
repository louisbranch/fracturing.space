package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListLocaleDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "pt-BR"), 0o755); err != nil {
		t.Fatalf("mkdir pt-BR: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "en-US"), 0o755); err != nil {
		t.Fatalf("mkdir en-US: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	locales, err := listLocaleDirs(root)
	if err != nil {
		t.Fatalf("listLocaleDirs returned error: %v", err)
	}
	expected := []string{"en-US", "pt-BR"}
	if strings.Join(locales, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected %v, got %v", expected, locales)
	}
}

func TestReadJSONMissingFile(t *testing.T) {
	root := t.TempDir()
	got, err := readJSON[classPayload](root, "classes.json")
	if err != nil {
		t.Fatalf("readJSON returned error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil payload for missing file")
	}
}

func TestReadJSONInvalid(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "classes.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("write classes.json: %v", err)
	}
	_, err := readJSON[classPayload](root, "classes.json")
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
	if !strings.Contains(err.Error(), "decode classes.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLocalePayloads(t *testing.T) {
	locale := "en-US"
	payloads := localePayloads{
		Classes: &classPayload{
			SystemID:      defaultSystemID,
			SystemVersion: defaultSystemVer,
			Source:        "source",
			Locale:        locale,
		},
	}
	if err := validateLocalePayloads(locale, payloads); err != nil {
		t.Fatalf("expected payloads to be valid: %v", err)
	}

	payloads.Classes.SystemID = "other"
	if err := validateLocalePayloads(locale, payloads); err == nil {
		t.Fatal("expected error for unsupported system id")
	}
}

func TestContains(t *testing.T) {
	items := []string{"a", "b"}
	if !contains(items, "b") {
		t.Fatal("expected contains to return true")
	}
	if contains(items, "c") {
		t.Fatal("expected contains to return false")
	}
}

func TestUpsertLocaleRequiresStore(t *testing.T) {
	err := upsertLocale(nil, nil, "en-US", true, localePayloads{}, time.Now())
	if err == nil {
		t.Fatal("expected error when content store is nil")
	}
}
