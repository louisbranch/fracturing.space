package game

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

func TestDirectAppendEventUsageIsRestrictedToMaintenancePaths(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	grpcRoot := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc")
	allowed := map[string]struct{}{
		"internal/services/game/api/grpc/game/domain_adapter.go":    {},
		"internal/services/game/api/grpc/game/event_application.go": {},
		"internal/services/game/api/grpc/game/fork_application.go":  {},
	}

	var violations []string
	walkErr := filepath.WalkDir(grpcRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if _, ok := allowed[relPath]; ok {
			return nil
		}
		lines, err := appendEventCallLines(path)
		if err != nil {
			return err
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", relPath, line))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan grpc files: %v", walkErr)
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("direct AppendEvent usage outside maintenance/import paths:\n%s", strings.Join(violations, "\n"))
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", ".."))
}

func appendEventCallLines(path string) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	lines := make([]int, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel == nil || sel.Sel.Name != "AppendEvent" {
			return true
		}
		parentSelector, ok := sel.X.(*ast.SelectorExpr)
		if !ok || parentSelector.Sel == nil || parentSelector.Sel.Name != "Event" {
			return true
		}
		line := fset.Position(sel.Sel.Pos()).Line
		lines = append(lines, line)
		return true
	})
	return lines, nil
}
