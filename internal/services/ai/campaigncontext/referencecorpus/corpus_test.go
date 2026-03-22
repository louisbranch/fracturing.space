package referencecorpus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCorpusSearchAndRead(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, `[
  {"id":"moves","title":"Fear Moves","kind":"rule","path":"moves.md","aliases":["gm moves"]},
  {"id":"combat","title":"Combat Flow","kind":"guide","path":"combat.md","aliases":["initiative"]}
]`)
	writeFile(t, root, "moves.md", "# Fear Moves\nEscalate fear when the table stalls.")
	writeFile(t, root, "combat.md", "# Combat Flow\nUse action tracker and spotlight order.")

	corpus := New(root)
	results, err := corpus.Search(context.Background(), supportedSystem, "gm moves", 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 || results[0].DocumentID != "moves" {
		t.Fatalf("Search() results = %+v, want moves first", results)
	}
	if !strings.Contains(results[0].Snippet, "aliases: gm moves") {
		t.Fatalf("metadata snippet = %q, want alias snippet", results[0].Snippet)
	}

	results, err = corpus.Search(context.Background(), supportedSystem, "action tracker", 1)
	if err != nil {
		t.Fatalf("Search(content fallback) error = %v", err)
	}
	if len(results) != 1 || results[0].DocumentID != "combat" {
		t.Fatalf("Search(content fallback) results = %+v, want combat", results)
	}
	if !strings.Contains(strings.ToLower(results[0].Snippet), "spotlight") {
		t.Fatalf("content snippet = %q, want spotlight", results[0].Snippet)
	}

	document, err := corpus.Read(context.Background(), supportedSystem, "combat")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !strings.Contains(document.Content, "spotlight order") {
		t.Fatalf("document content = %q, want combat content", document.Content)
	}

	document, err = corpus.Read(context.Background(), supportedSystem, "moves.md")
	if err != nil {
		t.Fatalf("Read(path) error = %v", err)
	}
	if document.DocumentID != "moves" {
		t.Fatalf("document id = %q, want %q", document.DocumentID, "moves")
	}
}

func TestCorpusValidationAndPathSafety(t *testing.T) {
	corpus := New("")
	if _, err := corpus.Search(context.Background(), supportedSystem, "fear", 0); err == nil || !strings.Contains(err.Error(), "reference root is not configured") {
		t.Fatalf("Search() error = %v, want missing root", err)
	}
	if _, err := corpus.Search(context.Background(), "other", "fear", 0); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("Search(unsupported) error = %v", err)
	}
	if _, err := corpus.Search(context.Background(), supportedSystem, " ", 0); err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("Search(blank query) error = %v", err)
	}
	if _, err := corpus.Read(context.Background(), supportedSystem, " "); err == nil || !strings.Contains(err.Error(), "document id is required") {
		t.Fatalf("Read(blank document id) error = %v", err)
	}

	root := t.TempDir()
	writeFixture(t, root, `[{"id":"escape","title":"Escape","kind":"rule","path":"../escape.md"}]`)
	corpus = New(root)
	if _, err := corpus.Read(context.Background(), supportedSystem, "escape"); err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("Read(escape) error = %v, want escape error", err)
	}
}

func TestCorpusIncludesRepoPlaybooks(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, `[]`)

	corpus := New(root)
	results, err := corpus.Search(context.Background(), supportedSystem, "spotlight combat", 5)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	found := false
	for _, result := range results {
		if result.DocumentID == "playbook-gm-fear-adversaries-and-spotlight" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Search() results = %+v, want repo playbook hit", results)
	}

	document, err := corpus.Read(context.Background(), supportedSystem, "playbook-combat-procedures")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !strings.Contains(document.Content, "daggerheart_reaction_flow_resolve") {
		t.Fatalf("document content = %q, want reaction flow guidance", document.Content)
	}
}

func writeFixture(t *testing.T, root, contents string) {
	t.Helper()
	writeFile(t, root, "index.json", contents)
}

func writeFile(t *testing.T, root, path, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", fullPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", fullPath, err)
	}
}
