package daggerheart

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module/testkit"
)

func TestDaggerheartHandlersUseSharedDomainWriteHelper(t *testing.T) {
	dir := handlerDir(t)
	files, err := testkit.ScanHandlerDir(dir)
	if err != nil {
		t.Fatalf("load architecture scan files: %v", err)
	}

	var violations []string
	for _, filename := range files {
		sourcePath := filepath.Join(dir, filename)
		domainAliases, err := findDomainStoreAliases(sourcePath)
		if err != nil {
			t.Fatalf("scan aliases in %s: %v", filename, err)
		}
		lines, err := testkit.ScanCallViolations(sourcePath, func(callPath string) bool {
			if callPath == "s.stores.Write.Executor.Execute" {
				return true
			}
			for alias := range domainAliases {
				if callPath == alias+".Execute" {
					return true
				}
			}
			return false
		})
		if err != nil {
			t.Fatalf("scan %s: %v", filename, err)
		}
		violations = append(violations, linesWithFile(filename, lines)...)
	}

	assertNoViolations(t, "direct Domain.Execute usage found in Daggerheart handlers", violations)
}

// TestDaggerheartWritePathArchitecture uses the generalized write-path guard
// to enforce: no inline Apply calls, no direct storage mutations, and no
// forbidden string literals.
func TestDaggerheartWritePathArchitecture(t *testing.T) {
	testkit.ValidateWritePathArchitecture(t, testkit.WritePathPolicy{
		HandlerDir: handlerDir(t),
		StoreMutationSubstrings: []string{
			".PutDaggerheart",
			".UpdateDaggerheart",
			".DeleteDaggerheart",
		},
		LiteralPolicies: map[string][]string{
			"actions.go": {
				"action.outcome_rejected",
				"story.note_added",
			},
		},
	})
}

func TestDaggerheartArchScanIncludesNonLegacyFiles(t *testing.T) {
	dir := handlerDir(t)
	files, err := testkit.ScanHandlerDir(dir)
	if err != nil {
		t.Fatalf("load architecture scan files: %v", err)
	}
	if !containsFile(files, "workflow_condition_handler.go") {
		t.Fatal("expected architecture scan to include workflow_condition_handler.go")
	}
}

func TestDaggerheartPackageDoesNotImportRootGameTransport(t *testing.T) {
	dir := handlerDir(t)
	files, err := testkit.ScanHandlerDir(dir)
	if err != nil {
		t.Fatalf("load architecture scan files: %v", err)
	}

	var violations []string
	for _, filename := range files {
		path := filepath.Join(dir, filename)
		imports, err := importPaths(path)
		if err != nil {
			t.Fatalf("scan imports in %s: %v", filename, err)
		}
		for _, importPath := range imports {
			if importPath == "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game" {
				violations = append(violations, filename)
				break
			}
		}
	}

	assertNoViolations(t, "Daggerheart transport imports the root game transport package", violations)
}

// handlerDir resolves the directory containing this test file at runtime.
func handlerDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}
	return filepath.Dir(thisFile)
}

// findDomainStoreAliases detects local variables assigned from s.stores.Write.Executor
// so the shared-helper guard catches aliased calls too.
func findDomainStoreAliases(path string) (map[string]struct{}, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	aliases := make(map[string]struct{})
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			for i, rhs := range typed.Rhs {
				if !isDomainStoreSelector(rhs) || i >= len(typed.Lhs) {
					continue
				}
				ident, ok := typed.Lhs[i].(*ast.Ident)
				if !ok || ident.Name == "_" {
					continue
				}
				aliases[ident.Name] = struct{}{}
			}
		case *ast.ValueSpec:
			for i, rhs := range typed.Values {
				if !isDomainStoreSelector(rhs) || i >= len(typed.Names) {
					continue
				}
				name := typed.Names[i].Name
				if name != "_" {
					aliases[name] = struct{}{}
				}
			}
		}
		return true
	})
	return aliases, nil
}

func isDomainStoreSelector(expr ast.Expr) bool {
	return selectorPathLocal(expr) == "s.stores.Write.Executor"
}

// selectorPathLocal resolves the dot-separated selector path. This is the
// package-local version needed for alias detection; the shared version lives
// in testkit.
func selectorPathLocal(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		prefix := selectorPathLocal(typed.X)
		if prefix == "" {
			return typed.Sel.Name
		}
		return prefix + "." + typed.Sel.Name
	case *ast.Ident:
		return typed.Name
	case *ast.ParenExpr:
		return selectorPathLocal(typed.X)
	case *ast.StarExpr:
		return selectorPathLocal(typed.X)
	default:
		return ""
	}
}

func linesWithFile(filename string, lines []int) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, fmt.Sprintf("%s:%d", filename, line))
	}
	return out
}

func containsFile(files []string, target string) bool {
	for _, file := range files {
		if file == target {
			return true
		}
	}
	return false
}

func importPaths(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, "\"")
		if importPath != "" {
			imports = append(imports, importPath)
		}
	}
	return imports, nil
}

func assertNoViolations(t *testing.T, message string, violations []string) {
	t.Helper()
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("%s:\n%s", message, strings.Join(violations, "\n"))
}
