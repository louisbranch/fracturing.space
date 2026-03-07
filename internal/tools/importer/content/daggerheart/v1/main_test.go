package catalogimporter

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
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

func TestRunSkipIfReadySkipsWritesWhenCatalogAlreadyReady(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "content.db")
	cfg := Config{
		Dir:        ".",
		DBPath:     dbPath,
		BaseLocale: defaultBaseLocale,
	}

	if err := Run(context.Background(), cfg, io.Discard); err != nil {
		t.Fatalf("initial Run() error = %v", err)
	}

	store, err := storagesqlite.OpenContent(dbPath)
	if err != nil {
		t.Fatalf("OpenContent() error = %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}()

	before, err := store.GetDaggerheartClass(context.Background(), "class.guardian")
	if err != nil {
		t.Fatalf("GetDaggerheartClass(before) error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	out := &bytes.Buffer{}
	skipCfg := cfg
	skipCfg.SkipIfReady = true
	if err := Run(context.Background(), skipCfg, out); err != nil {
		t.Fatalf("skip Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "catalog already ready") {
		t.Fatalf("output = %q, want readiness skip message", out.String())
	}

	after, err := store.GetDaggerheartClass(context.Background(), "class.guardian")
	if err != nil {
		t.Fatalf("GetDaggerheartClass(after) error = %v", err)
	}
	if !before.UpdatedAt.Equal(after.UpdatedAt) {
		t.Fatalf("UpdatedAt changed on skipped run: before=%v after=%v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestRunSkipIfReadyImportsWhenCatalogNotReady(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "content.db")
	out := &bytes.Buffer{}
	cfg := Config{
		Dir:         ".",
		DBPath:      dbPath,
		BaseLocale:  defaultBaseLocale,
		SkipIfReady: true,
	}

	if err := Run(context.Background(), cfg, out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "imported") {
		t.Fatalf("output = %q, want import message", out.String())
	}

	store, err := storagesqlite.OpenContent(dbPath)
	if err != nil {
		t.Fatalf("OpenContent() error = %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}()

	readiness, err := storage.EvaluateDaggerheartCatalogReadiness(context.Background(), store)
	if err != nil {
		t.Fatalf("EvaluateDaggerheartCatalogReadiness() error = %v", err)
	}
	if !readiness.Ready {
		t.Fatalf("readiness.Ready = false, missing %v", readiness.MissingSections)
	}
}
