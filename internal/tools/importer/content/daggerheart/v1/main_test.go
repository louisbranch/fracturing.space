package catalogimporter

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
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

func TestRunWithDepsDryRunSkipsStoreOpen(t *testing.T) {
	openCalls := 0
	deps := runDeps{
		openStore: func(string) (contentStore, error) {
			openCalls++
			return nil, errors.New("open should not be called during dry-run")
		},
		nowUTC: func() time.Time {
			return time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		},
	}

	cfg := Config{
		Dir:        ".",
		BaseLocale: defaultBaseLocale,
		DryRun:     true,
	}
	var out bytes.Buffer
	if err := runWithDeps(context.Background(), cfg, &out, deps); err != nil {
		t.Fatalf("runWithDeps() error = %v", err)
	}
	if openCalls != 0 {
		t.Fatalf("open store calls = %d, want 0", openCalls)
	}
	if !strings.Contains(out.String(), "validated") {
		t.Fatalf("output = %q, want validation summary", out.String())
	}
}

func TestRunWithDepsFixtureValidationError(t *testing.T) {
	root := t.TempDir()
	localeDir := filepath.Join(root, defaultBaseLocale)
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir locale: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(localeDir, "classes.json"),
		[]byte(`{"system_id":"other","system_version":"v1","source":"seed","locale":"en-US","items":[]}`),
		0o644,
	); err != nil {
		t.Fatalf("write classes fixture: %v", err)
	}

	cfg := Config{
		Dir:        root,
		BaseLocale: defaultBaseLocale,
		DryRun:     true,
	}
	err := runWithDeps(context.Background(), cfg, io.Discard, runDeps{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "validate en-US: unsupported system id other") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenContentStoreWithRetryRetriesBusyAndSucceeds(t *testing.T) {
	busyErr := generateBusySQLiteError(t)
	successPath := filepath.Join(t.TempDir(), "content.db")

	var attempts int
	openStore := func(string) (contentStore, error) {
		attempts++
		if attempts < 3 {
			return nil, busyErr
		}
		return storagesqlite.OpenContent(successPath)
	}

	store, err := openContentStoreWithRetry(context.Background(), successPath, io.Discard, openStore)
	if err != nil {
		t.Fatalf("openContentStoreWithRetry() error = %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}()
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestOpenContentStoreWithRetryDoesNotRetryNonBusy(t *testing.T) {
	var attempts int
	openStore := func(string) (contentStore, error) {
		attempts++
		return nil, errors.New("boom")
	}

	_, err := openContentStoreWithRetry(context.Background(), filepath.Join(t.TempDir(), "content.db"), io.Discard, openStore)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestOpenContentStoreWithRetryRespectsContextCancellation(t *testing.T) {
	busyErr := generateBusySQLiteError(t)
	openStore := func(string) (contentStore, error) {
		return nil, busyErr
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := openContentStoreWithRetry(ctx, filepath.Join(t.TempDir(), "content.db"), io.Discard, openStore)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func generateBusySQLiteError(t *testing.T) error {
	t.Helper()

	path := filepath.Join(t.TempDir(), "busy.db")
	dsn := path + "?_pragma=busy_timeout(0)"

	db1, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}
	defer db1.Close()

	db2, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()

	if _, err := db1.Exec("CREATE TABLE locks (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	tx, err := db1.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("INSERT INTO locks (id) VALUES (1)"); err != nil {
		t.Fatalf("insert in tx: %v", err)
	}

	_, busyErr := db2.Exec("INSERT INTO locks (id) VALUES (2)")
	if busyErr == nil {
		t.Fatal("expected busy/locked error")
	}

	return busyErr
}
