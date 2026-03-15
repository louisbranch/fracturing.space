package campaigncontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceCorpusSearchAndRead(t *testing.T) {
	root := t.TempDir()
	writeReferenceFixture(t, root, `[
  {"id":"moves","title":"Fear Moves","kind":"rule","path":"moves.md","aliases":["gm moves"]},
  {"id":"combat","title":"Combat Flow","kind":"guide","path":"combat.md","aliases":["initiative"]}
]`)
	writeReferenceFile(t, root, "moves.md", "# Fear Moves\nEscalate fear when the table stalls.")
	writeReferenceFile(t, root, "combat.md", "# Combat Flow\nUse action tracker and spotlight order.")

	corpus := NewReferenceCorpus(root)
	results, err := corpus.Search(context.Background(), DaggerheartSystem, "gm moves", 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 || results[0].DocumentID != "moves" {
		t.Fatalf("Search() results = %+v, want moves first", results)
	}
	if !strings.Contains(results[0].Snippet, "aliases: gm moves") {
		t.Fatalf("metadata snippet = %q, want alias snippet", results[0].Snippet)
	}

	results, err = corpus.Search(context.Background(), DaggerheartSystem, "spotlight", 1)
	if err != nil {
		t.Fatalf("Search(content fallback) error = %v", err)
	}
	if len(results) != 1 || results[0].DocumentID != "combat" {
		t.Fatalf("Search(content fallback) results = %+v, want combat", results)
	}
	if !strings.Contains(strings.ToLower(results[0].Snippet), "spotlight") {
		t.Fatalf("content snippet = %q, want spotlight", results[0].Snippet)
	}

	document, err := corpus.Read(context.Background(), DaggerheartSystem, "combat")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !strings.Contains(document.Content, "spotlight order") {
		t.Fatalf("document content = %q, want combat content", document.Content)
	}

	document, err = corpus.Read(context.Background(), DaggerheartSystem, "moves.md")
	if err != nil {
		t.Fatalf("Read(path) error = %v", err)
	}
	if document.DocumentID != "moves" {
		t.Fatalf("document id = %q, want %q", document.DocumentID, "moves")
	}
}

func TestReferenceCorpusValidationAndPathSafety(t *testing.T) {
	corpus := NewReferenceCorpus("")
	if _, err := corpus.Search(context.Background(), DaggerheartSystem, "fear", 0); err == nil || !strings.Contains(err.Error(), "reference root is not configured") {
		t.Fatalf("Search() error = %v, want missing root", err)
	}
	if _, err := corpus.Search(context.Background(), "other", "fear", 0); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("Search(unsupported) error = %v", err)
	}
	if _, err := corpus.Search(context.Background(), DaggerheartSystem, " ", 0); err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("Search(blank query) error = %v", err)
	}
	if _, err := corpus.Read(context.Background(), DaggerheartSystem, " "); err == nil || !strings.Contains(err.Error(), "document id is required") {
		t.Fatalf("Read(blank document id) error = %v", err)
	}

	root := t.TempDir()
	writeReferenceFixture(t, root, `[{"id":"escape","title":"Escape","kind":"rule","path":"../escape.md"}]`)
	corpus = NewReferenceCorpus(root)
	if _, err := corpus.Read(context.Background(), DaggerheartSystem, "escape"); err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("Read(escape) error = %v, want escape error", err)
	}
}

func writeReferenceFixture(t *testing.T, root string, contents string) {
	t.Helper()
	writeReferenceFile(t, root, "index.json", contents)
}

func writeReferenceFile(t *testing.T, root string, path string, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", fullPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", fullPath, err)
	}
}
