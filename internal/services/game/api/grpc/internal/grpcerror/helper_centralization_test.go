package grpcerror

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestDomainWriteStatusHelpersStayCentralized(t *testing.T) {
	repoRoot := findRepoRoot(t)
	grpcRoot := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc")
	disallowed := map[string]struct{}{
		"ensureGRPCStatus":      {},
		"normalizeGRPCDefaults": {},
	}

	var violations []string
	err := filepath.WalkDir(grpcRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil {
				continue
			}
			if _, banned := disallowed[fn.Name.Name]; !banned {
				continue
			}
			violations = append(violations, fmt.Sprintf("%s:%d %s", rel, fset.Position(fn.Pos()).Line, fn.Name.Name))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan grpc tree: %v", err)
	}
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("duplicate status helper implementations detected:\n%s", strings.Join(violations, "\n"))
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found when walking to repo root")
		}
		dir = parent
	}
}
