package app

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAppPackageDoesNotConstructRuntimeInfrastructure(t *testing.T) {
	t.Parallel()

	root := appPackageRoot(t)
	for _, entry := range goFilesInDir(t, root) {
		parsed, err := parser.ParseFile(token.NewFileSet(), entry, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", entry, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			switch path {
			case "github.com/louisbranch/fracturing.space/internal/platform/grpc":
				t.Fatalf("%s imports %q; keep connection construction in internal/cmd/play", entry, path)
			case "github.com/louisbranch/fracturing.space/internal/services/play/storage/sqlite":
				t.Fatalf("%s imports %q; keep storage construction in internal/cmd/play", entry, path)
			}
		}
	}
}

func appPackageRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	return filepath.Dir(file)
}

func goFilesInDir(t *testing.T, root string) []string {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read dir %s: %v", root, err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if entry.IsDir() || name == "" || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		paths = append(paths, filepath.Join(root, name))
	}
	return paths
}
