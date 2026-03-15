package web

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestDependenciesDelegateBindingWithoutGeneratedClientImports(t *testing.T) {
	t.Parallel()

	assertImportsDoNotContainPrefix(t, "dependencies.go", "github.com/louisbranch/fracturing.space/api/gen/go/")

	for _, tc := range []struct {
		funcName string
		want     []selectorCall
	}{
		{
			funcName: "BindAuthDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindAuthDependency"},
				{recv: "modules", name: "BindAuthDependency"},
			},
		},
		{
			funcName: "BindSocialDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindSocialDependency"},
				{recv: "modules", name: "BindSocialDependency"},
			},
		},
		{
			funcName: "BindGameDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindGameDependency"},
			},
		},
		{
			funcName: "BindAIDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindAIDependency"},
			},
		},
		{
			funcName: "BindDiscoveryDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindDiscoveryDependency"},
			},
		},
		{
			funcName: "BindUserHubDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindUserHubDependency"},
			},
		},
		{
			funcName: "BindNotificationsDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindNotificationsDependency"},
				{recv: "modules", name: "BindNotificationsDependency"},
			},
		},
		{
			funcName: "BindStatusDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindStatusDependency"},
			},
		},
	} {
		for _, call := range tc.want {
			assertFuncCallsSelector(t, "dependencies.go", tc.funcName, call.recv, call.name)
		}
	}
}

func TestStartupDependencyDescriptorsStayServiceOwnedAndBinderBased(t *testing.T) {
	t.Parallel()

	assertImportsDoNotContainPrefix(t, "startup_dependencies.go", "github.com/louisbranch/fracturing.space/api/gen/go/")
	for _, typeName := range []string{
		"StartupDependencyDescriptor",
		"StartupDependencyPolicy",
	} {
		assertTypeExists(t, "startup_dependencies.go", typeName)
	}
	for _, funcName := range []string{
		"StartupDependencyDescriptors",
		"LookupStartupDependencyDescriptor",
	} {
		assertFuncExists(t, "startup_dependencies.go", funcName)
	}
}

type selectorCall struct {
	recv string
	name string
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

func assertFuncExists(t *testing.T, path, funcName string) {
	t.Helper()

	parsed := parseFile(t, path)
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name != nil && fn.Name.Name == funcName {
			return
		}
	}
	t.Fatalf("%s missing func %s", path, funcName)
}

func assertTypeExists(t *testing.T, path, typeName string) {
	t.Helper()

	parsed := parseFile(t, path)
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name != nil && typeSpec.Name.Name == typeName {
				return
			}
		}
	}
	t.Fatalf("%s missing type %s", path, typeName)
}

func assertImportsDoNotContainPrefix(t *testing.T, path, prefix string) {
	t.Helper()

	parsed := parseFile(t, path)
	for _, imp := range parsed.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		if strings.HasPrefix(importPath, prefix) {
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
