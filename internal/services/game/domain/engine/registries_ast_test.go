package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// serviceRoot returns the absolute path to the game service root by walking up
// from this test file's location.
func serviceRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// thisFile is .../domain/engine/registries_ast_test.go
	// service root is .../game/
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// parseFuncBody parses a Go file and returns the body of the named function.
func parseFuncBody(t *testing.T, filePath, funcName string) *ast.BlockStmt {
	t.Helper()
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse %s: %v", filePath, err)
	}
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == funcName {
			return fn.Body
		}
	}
	t.Fatalf("function %s not found in %s", funcName, filePath)
	return nil
}

// extractSliceLiteralIdents extracts all identifiers from the first composite
// literal (slice literal) found in a return statement within body. This
// captures the list returned by FoldHandledTypes() / ProjectionHandledTypes().
func extractSliceLiteralIdents(body *ast.BlockStmt) []string {
	var idents []string
	ast.Inspect(body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, result := range ret.Results {
			comp, ok := result.(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, elt := range comp.Elts {
				switch v := elt.(type) {
				case *ast.Ident:
					idents = append(idents, v.Name)
				case *ast.SelectorExpr:
					idents = append(idents, selectorString(v))
				}
			}
		}
		return true
	})
	return idents
}

// extractSwitchCaseIdents extracts event type identifiers from switch cases
// that switch on a selector like evt.Type.
func extractSwitchCaseIdents(body *ast.BlockStmt) []string {
	var idents []string
	ast.Inspect(body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		// Verify the switch tag references .Type
		if !isSelectorNamed(sw.Tag, "Type") {
			return true
		}
		for _, stmt := range sw.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok || cc.List == nil { // skip default
				continue
			}
			for _, expr := range cc.List {
				switch v := expr.(type) {
				case *ast.Ident:
					idents = append(idents, v.Name)
				case *ast.SelectorExpr:
					idents = append(idents, selectorString(v))
				}
			}
		}
		return true
	})
	return idents
}

// extractIfConditionIdents extracts event type identifiers from
// `if evt.Type == SomeConst` conditions, including OR conditions like
// `if evt.Type == A || evt.Type == B`.
func extractIfConditionIdents(body *ast.BlockStmt) []string {
	var idents []string
	ast.Inspect(body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		idents = append(idents, extractTypeComparisonsFromExpr(ifStmt.Cond)...)
		return true
	})
	return idents
}

// extractTypeComparisonsFromExpr recursively extracts event type identifiers
// from binary expressions like `evt.Type == X || evt.Type == Y`.
func extractTypeComparisonsFromExpr(expr ast.Expr) []string {
	var idents []string
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if e.Op == token.LOR {
			// OR expression: recurse both sides
			idents = append(idents, extractTypeComparisonsFromExpr(e.X)...)
			idents = append(idents, extractTypeComparisonsFromExpr(e.Y)...)
		} else if e.Op == token.EQL {
			// Equality: check if left side is evt.Type
			if isSelectorNamed(e.X, "Type") {
				switch v := e.Y.(type) {
				case *ast.Ident:
					idents = append(idents, v.Name)
				case *ast.SelectorExpr:
					idents = append(idents, selectorString(v))
				}
			}
		}
	}
	return idents
}

// selectorString returns "X.Sel" for a selector expression.
func selectorString(sel *ast.SelectorExpr) string {
	if id, ok := sel.X.(*ast.Ident); ok {
		return id.Name + "." + sel.Sel.Name
	}
	return sel.Sel.Name
}

// isSelectorNamed returns true if expr is a selector ending in the given name.
func isSelectorNamed(expr ast.Expr, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == name
}

// sortedUnique returns a sorted, deduplicated copy of the input strings.
func sortedUnique(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	var out []string
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func TestFoldHandledTypes_MatchActualFoldCases(t *testing.T) {
	root := serviceRoot(t)
	domainDir := filepath.Join(root, "domain")

	tests := []struct {
		name      string
		pkg       string
		useSwitch bool // true = switch evt.Type, false = if evt.Type ==
	}{
		{"campaign", "campaign", false},
		{"session", "session", false},
		{"action", "action", true},
		{"participant", "participant", false},
		{"character", "character", false},
		{"invite", "invite", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foldFile := filepath.Join(domainDir, tt.pkg, "fold.go")
			if _, err := os.Stat(foldFile); err != nil {
				t.Fatalf("fold file not found: %s", foldFile)
			}

			// Extract types declared in FoldHandledTypes().
			handledBody := parseFuncBody(t, foldFile, "FoldHandledTypes")
			declared := sortedUnique(extractSliceLiteralIdents(handledBody))

			// Extract types referenced in Fold() body.
			foldBody := parseFuncBody(t, foldFile, "Fold")
			var referenced []string
			if tt.useSwitch {
				referenced = sortedUnique(extractSwitchCaseIdents(foldBody))
			} else {
				referenced = sortedUnique(extractIfConditionIdents(foldBody))
			}

			// declared must be a superset of referenced (fold can declare
			// types that are deliberately no-ops, like campaign.EventTypeForked)
			refSet := make(map[string]struct{}, len(referenced))
			for _, r := range referenced {
				refSet[r] = struct{}{}
			}

			// Every referenced type must be declared.
			for _, r := range referenced {
				found := false
				for _, d := range declared {
					if d == r {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("fold references %s but FoldHandledTypes() does not declare it", r)
				}
			}

			// Every declared type must be referenced (unless deliberately no-op).
			for _, d := range declared {
				if _, ok := refSet[d]; !ok {
					// This is intentional for some types (e.g., campaign.EventTypeForked)
					// but we still flag it as informational.
					t.Logf("INFO: FoldHandledTypes() declares %s but no fold case references it (may be intentional no-op)", d)
				}
			}
		})
	}
}

// TestProjectionHandledTypes_NonEmpty validates that the projection handler
// registry contains entries. This replaced the former AST-based test that
// verified a switch statement; the switch was replaced with a handler map
// in the projection package, so the AST approach is no longer applicable.
// Runtime validation lives in projection/handler_registry_test.go.
func TestProjectionHandledTypes_NonEmpty(t *testing.T) {
	types := projection.ProjectionHandledTypes()
	if len(types) == 0 {
		t.Fatal("ProjectionHandledTypes() returned empty list — handler registry may be broken")
	}
	seen := make(map[event.Type]struct{}, len(types))
	for _, et := range types {
		if _, dup := seen[et]; dup {
			t.Errorf("duplicate type in ProjectionHandledTypes(): %s", et)
		}
		seen[et] = struct{}{}
	}
	t.Logf("projection: %d handler types registered", len(types))
}

// TestFoldAndProjectionFiles_Exist ensures the source files we parse are present.
// If the project structure changes, this test will fail early with a clear message.
func TestFoldAndProjectionFiles_Exist(t *testing.T) {
	root := serviceRoot(t)
	files := []string{
		filepath.Join(root, "domain", "campaign", "fold.go"),
		filepath.Join(root, "domain", "session", "fold.go"),
		filepath.Join(root, "domain", "action", "fold.go"),
		filepath.Join(root, "domain", "participant", "fold.go"),
		filepath.Join(root, "domain", "character", "fold.go"),
		filepath.Join(root, "domain", "invite", "fold.go"),
		filepath.Join(root, "projection", "apply_campaign.go"),
		filepath.Join(root, "projection", "apply_participant.go"),
		filepath.Join(root, "projection", "apply_character.go"),
		filepath.Join(root, "projection", "apply_invite.go"),
		filepath.Join(root, "projection", "apply_session.go"),
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			// Use a relative path fragment for readability.
			rel := strings.TrimPrefix(f, root+"/")
			t.Errorf("expected file %s to exist", rel)
		}
	}
}
