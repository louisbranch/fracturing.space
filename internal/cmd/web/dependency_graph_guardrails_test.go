package web

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestDependencyGraphUsesServiceOwnedDescriptorsAndOnlyStatusProto(t *testing.T) {
	t.Parallel()

	assertImportPresent(t, "dependency_graph.go", "github.com/louisbranch/fracturing.space/internal/services/web")
	assertImportPresent(t, "dependency_graph.go", "github.com/louisbranch/fracturing.space/api/gen/go/status/v1")
	assertImportsDoNotContainPrefixExcept(t, "dependency_graph.go", "github.com/louisbranch/fracturing.space/api/gen/go/", "github.com/louisbranch/fracturing.space/api/gen/go/status/v1")
	assertFuncCallsSelector(t, "dependency_graph.go", "dependencyRequirements", "web", "StartupDependencyDescriptors")
}

func assertFuncCallsSelector(t *testing.T, path, funcName, recvName, selector string) {
	t.Helper()

	parsed := parseFile(t, path)
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != funcName || fn.Body == nil {
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != selector {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != recvName {
				return true
			}
			found = true
			return false
		})
		if !found {
			t.Fatalf("%s %s does not call %s.%s", path, funcName, recvName, selector)
		}
		return
	}

	t.Fatalf("%s missing func %s", path, funcName)
}

func assertImportPresent(t *testing.T, path, importPath string) {
	t.Helper()

	parsed := parseFile(t, path)
	for _, imp := range parsed.Imports {
		if strings.Trim(imp.Path.Value, "\"") == importPath {
			return
		}
	}
	t.Fatalf("%s missing import %s", path, importPath)
}

func assertImportsDoNotContainPrefixExcept(t *testing.T, path, prefix, allowed string) {
	t.Helper()

	parsed := parseFile(t, path)
	for _, imp := range parsed.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		if strings.HasPrefix(importPath, prefix) && importPath != allowed {
			t.Fatalf("%s unexpectedly imports %s", path, importPath)
		}
	}
}

func parseFile(t *testing.T, path string) *ast.File {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return parsed
}
