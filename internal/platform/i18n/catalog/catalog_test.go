package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmbeddedHasExpectedLocales(t *testing.T) {
	bundle, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load embedded catalogs: %v", err)
	}
	if !bundle.HasLocale(BaseLocale) {
		t.Fatalf("expected base locale %s", BaseLocale)
	}
	if !bundle.HasLocale("pt-BR") {
		t.Fatalf("expected locale pt-BR")
	}

	if got := len(bundle.LocaleMessages("en-US")); got == 0 {
		t.Fatalf("expected en-US messages")
	}
	if got := len(bundle.NamespaceMessages("en-US", "core")); got == 0 {
		t.Fatalf("expected en-US core namespace messages")
	}
}

func TestLoadFromFSRejectsCoreKeyOutsideCoreNamespace(t *testing.T) {
	tempDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tempDir, "locales/en-US/web.yaml"), `locale: "en-US"
namespace: "web"
messages:
  "core.bad": "nope"
`)
	mustWriteFile(t, filepath.Join(tempDir, "locales/en-US/core.yaml"), `locale: "en-US"
namespace: "core"
messages:
  "core.good": "ok"
`)

	_, err := LoadFromFS(os.DirFS(tempDir))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadFromFSRejectsDuplicateKeysAcrossNamespaces(t *testing.T) {
	tempDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tempDir, "locales/en-US/core.yaml"), `locale: "en-US"
namespace: "core"
messages:
  "a.key": "a"
`)
	mustWriteFile(t, filepath.Join(tempDir, "locales/en-US/web.yaml"), `locale: "en-US"
namespace: "web"
messages:
  "a.key": "b"
`)

	_, err := LoadFromFS(os.DirFS(tempDir))
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func TestNamespaceMessagesWithFallback(t *testing.T) {
	bundle, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load embedded catalogs: %v", err)
	}
	resolved, messages := bundle.NamespaceMessagesWithFallback("fr-FR", "errors")
	if resolved != "en-US" {
		t.Fatalf("resolved locale = %q, want en-US", resolved)
	}
	if len(messages) == 0 {
		t.Fatal("expected fallback errors namespace messages")
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
