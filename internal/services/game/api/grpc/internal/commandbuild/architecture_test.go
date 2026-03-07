package commandbuild

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestCommandbuildDoesNotImportSystemSpecificPackages(t *testing.T) {
	root := commandbuildDir(t)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read commandbuild dir: %v", err)
	}

	var violations []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(root, name)
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, spec := range node.Imports {
			importPath := strings.Trim(spec.Path.Value, "\"")
			if strings.Contains(importPath, "/internal/services/game/domain/bridge/") {
				violations = append(violations, name+": "+importPath)
			}
		}
	}

	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("system-specific imports found in commandbuild:\n%s", strings.Join(violations, "\n"))
}

func commandbuildDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	return filepath.Clean(filepath.Dir(thisFile))
}
