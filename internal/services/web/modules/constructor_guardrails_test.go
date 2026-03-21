package modules

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/testast"
)

func TestModuleConstructorsUseCanonicalConfigContract(t *testing.T) {
	t.Parallel()

	checked := 0
	for _, moduleRoot := range discoverModules(t) {
		moduleFiles := goFilesUnder(t, moduleRoot, false)
		for _, path := range moduleFiles {
			if filepath.Base(path) != "module.go" {
				continue
			}
			checked++
			assertModuleConstructorContract(t, path)
		}
	}
	if checked == 0 {
		t.Fatalf("no module constructor files found")
	}
}

func assertModuleConstructorContract(t *testing.T, path string) {
	t.Helper()

	parsed := testast.ParseFile(t, path)
	hasConfigType := false
	hasConfigConstructor := false
	hasCanonicalConstructor := false

	for _, decl := range parsed.Decls {
		switch node := decl.(type) {
		case *ast.GenDecl:
			if node.Tok != token.TYPE {
				continue
			}
			for _, spec := range node.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name == nil || typeSpec.Name.Name != "Config" {
					continue
				}
				_, ok = typeSpec.Type.(*ast.StructType)
				if !ok {
					t.Fatalf("%s type Config must be a struct", path)
				}
				hasConfigType = true
			}
		case *ast.FuncDecl:
			if node.Name == nil {
				continue
			}
			if strings.HasPrefix(node.Name.Name, "NewWith") {
				t.Fatalf("%s defines legacy constructor %s; use New(Config) only", path, node.Name.Name)
			}
			if isModuleConstructor(node) {
				hasConfigConstructor = true
			}
			if node.Name.Name != "New" {
				continue
			}
			assertCanonicalConstructorSignature(t, path, node)
			hasCanonicalConstructor = true
		}
	}

	if !hasConfigType {
		t.Fatalf("%s missing Config struct constructor contract", path)
	}
	if isPublicAuthModule(path) {
		if !hasConfigConstructor {
			t.Fatalf("%s expected at least one canonical Config constructor (for example NewShell)", path)
		}
		return
	}
	if !hasCanonicalConstructor {
		t.Fatalf("%s missing New(Config) constructor", path)
	}
}

func isPublicAuthModule(path string) bool {
	path = filepath.ToSlash(path)
	return path == "publicauth/module.go" || path == "./publicauth/module.go" || strings.HasPrefix(path, "publicauth/")
}

func isModuleConstructor(node *ast.FuncDecl) bool {
	if node == nil || node.Type == nil {
		return false
	}
	return hasSingleConfigParam(node.Type.Params) && hasSingleModuleResult(node.Type.Results)
}

func assertCanonicalConstructorSignature(t *testing.T, path string, fn *ast.FuncDecl) {
	t.Helper()

	if !hasSingleConfigParam(fn.Type.Params) {
		t.Fatalf("%s New constructor must accept exactly one parameter of type Config", path)
	}
	param := fn.Type.Params.List[0]
	ident, ok := param.Type.(*ast.Ident)
	if !ok || ident.Name != "Config" {
		t.Fatalf("%s New constructor parameter must be Config", path)
	}

	if !hasSingleModuleResult(fn.Type.Results) {
		t.Fatalf("%s New constructor must return exactly one value of type Module", path)
	}
	resultIdent, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	if !ok || resultIdent.Name != "Module" {
		t.Fatalf("%s New constructor must return Module", path)
	}
}

func hasSingleConfigParam(params *ast.FieldList) bool {
	return params != nil && len(params.List) == 1 && isConfigType(params.List[0].Type)
}

func hasSingleModuleResult(results *ast.FieldList) bool {
	return results != nil && len(results.List) == 1 && isModuleType(results.List[0].Type)
}

func isConfigType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "Config"
}

func isModuleType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "Module"
}
