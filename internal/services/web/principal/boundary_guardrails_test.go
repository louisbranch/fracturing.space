package principal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestResolverKeepsSplitCollaborators(t *testing.T) {
	t.Parallel()

	parsed, err := parser.ParseFile(token.NewFileSet(), "contracts.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse contracts.go: %v", err)
	}

	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil || typeSpec.Name.Name != "Resolver" {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				t.Fatalf("Resolver is not a struct")
			}

			fields := map[string]struct{}{}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					fields[name.Name] = struct{}{}
				}
			}

			for _, required := range []string{"auth", "viewer", "language"} {
				if _, ok := fields[required]; !ok {
					t.Fatalf("Resolver missing collaborator field %q", required)
				}
			}
			if _, ok := fields["deps"]; ok {
				t.Fatalf("Resolver still has monolithic deps field")
			}
			return
		}
	}

	t.Fatal("contracts.go missing Resolver type")
}
