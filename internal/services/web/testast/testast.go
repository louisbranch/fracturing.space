// Package testast provides shared AST assertion helpers for guardrail tests
// across the web service. Import this package only in test files.
package testast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// ParseFile parses a Go source file for AST inspection.
func ParseFile(t *testing.T, path string) *ast.File {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return parsed
}

// AssertFuncExists fails if the named function is missing from the file.
func AssertFuncExists(t *testing.T, path, funcName string) {
	t.Helper()
	if !HasFunc(path, funcName) {
		t.Fatalf("%s missing func %s", path, funcName)
	}
}

// AssertFuncMissing fails if the named function exists in the file.
func AssertFuncMissing(t *testing.T, path, funcName string) {
	t.Helper()
	if HasFunc(path, funcName) {
		t.Fatalf("%s unexpectedly defines func %s", path, funcName)
	}
}

// HasFunc reports whether the named function exists in the file.
func HasFunc(path, funcName string) bool {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name != nil && fn.Name.Name == funcName {
			return true
		}
	}
	return false
}

// AssertTypeExists fails if the named type is missing from the file.
func AssertTypeExists(t *testing.T, path, typeName string) {
	t.Helper()
	if !HasType(path, typeName) {
		t.Fatalf("%s missing type %s", path, typeName)
	}
}

// AssertTypeMissing fails if the named type exists in the file.
func AssertTypeMissing(t *testing.T, path, typeName string) {
	t.Helper()
	if HasType(path, typeName) {
		t.Fatalf("%s unexpectedly defines type %s", path, typeName)
	}
}

// HasType reports whether the named type exists in the file.
func HasType(path, typeName string) bool {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name != nil && typeSpec.Name.Name == typeName {
				return true
			}
		}
	}
	return false
}

// AssertImportPresent fails if the import path is missing from the file.
func AssertImportPresent(t *testing.T, path, importPath string) {
	t.Helper()
	if !HasImport(t, path, importPath) {
		t.Fatalf("%s missing import %s", path, importPath)
	}
}

// AssertImportAbsent fails if the import path exists in the file.
func AssertImportAbsent(t *testing.T, path, importPath string) {
	t.Helper()
	if HasImport(t, path, importPath) {
		t.Fatalf("%s unexpectedly imports %s", path, importPath)
	}
}

// HasImport reports whether the file imports the given path.
func HasImport(t *testing.T, path, importPath string) bool {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports for %s: %v", path, err)
	}
	for _, imp := range parsed.Imports {
		if strings.Trim(imp.Path.Value, "\"") == importPath {
			return true
		}
	}
	return false
}

// AssertImportsDoNotContainPrefix fails if any import starts with the prefix.
func AssertImportsDoNotContainPrefix(t *testing.T, path, prefix string) {
	t.Helper()
	parsed := ParseFile(t, path)
	for _, imp := range parsed.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		if strings.HasPrefix(importPath, prefix) {
			t.Fatalf("%s unexpectedly imports %s", path, importPath)
		}
	}
}

// AssertImportsDoNotContainPrefixExcept fails if any import starts with the
// prefix except for the allowed path.
func AssertImportsDoNotContainPrefixExcept(t *testing.T, path, prefix, allowed string) {
	t.Helper()
	parsed := ParseFile(t, path)
	for _, imp := range parsed.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		if strings.HasPrefix(importPath, prefix) && importPath != allowed {
			t.Fatalf("%s unexpectedly imports %s", path, importPath)
		}
	}
}

// AssertFuncCallsSelector fails if the named function does not call recv.selector.
func AssertFuncCallsSelector(t *testing.T, path, funcName, recvName, selector string) {
	t.Helper()
	if !FuncCalls(path, funcName, func(call *ast.CallExpr) bool {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != selector {
			return false
		}
		ident, ok := sel.X.(*ast.Ident)
		return ok && ident.Name == recvName
	}) {
		t.Fatalf("%s %s does not call %s.%s", path, funcName, recvName, selector)
	}
}

// AssertFuncDoesNotCallSelector fails if the named function calls recv.selector.
func AssertFuncDoesNotCallSelector(t *testing.T, path, funcName, recvName, selector string) {
	t.Helper()
	if FuncCalls(path, funcName, func(call *ast.CallExpr) bool {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != selector {
			return false
		}
		ident, ok := sel.X.(*ast.Ident)
		return ok && ident.Name == recvName
	}) {
		t.Fatalf("%s %s still calls %s.%s", path, funcName, recvName, selector)
	}
}

// AssertFuncCallsIdent fails if the named function does not call the ident.
func AssertFuncCallsIdent(t *testing.T, path, funcName, identName string) {
	t.Helper()
	if !FuncCalls(path, funcName, func(call *ast.CallExpr) bool {
		ident, ok := call.Fun.(*ast.Ident)
		return ok && ident.Name == identName
	}) {
		t.Fatalf("%s %s does not call %s", path, funcName, identName)
	}
}

// AssertFileCallsSelector fails if no call in the file matches recv.selector.
func AssertFileCallsSelector(t *testing.T, path, recvName, selector string) {
	t.Helper()
	parsed := ParseFile(t, path)
	found := false
	ast.Inspect(parsed, func(n ast.Node) bool {
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
		t.Fatalf("%s does not call %s.%s", path, recvName, selector)
	}
}

// FuncCalls reports whether the named function contains a call matching the predicate.
func FuncCalls(path, funcName string, match func(*ast.CallExpr) bool) bool {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
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
			if match(call) {
				found = true
				return false
			}
			return true
		})
		return found
	}
	return false
}
