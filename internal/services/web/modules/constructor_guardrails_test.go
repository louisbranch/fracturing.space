package modules

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
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

	parsed := parseFile(t, path)
	hasConfigType := false
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
	if !hasCanonicalConstructor {
		t.Fatalf("%s missing New(Config) constructor", path)
	}
}

func assertCanonicalConstructorSignature(t *testing.T, path string, fn *ast.FuncDecl) {
	t.Helper()

	if fn.Type == nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		t.Fatalf("%s New constructor must accept exactly one parameter of type Config", path)
	}
	param := fn.Type.Params.List[0]
	ident, ok := param.Type.(*ast.Ident)
	if !ok || ident.Name != "Config" {
		t.Fatalf("%s New constructor parameter must be Config", path)
	}

	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		t.Fatalf("%s New constructor must return exactly one value of type Module", path)
	}
	resultIdent, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	if !ok || resultIdent.Name != "Module" {
		t.Fatalf("%s New constructor must return Module", path)
	}
}
