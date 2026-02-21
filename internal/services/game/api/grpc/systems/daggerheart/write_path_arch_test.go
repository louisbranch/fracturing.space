package daggerheart

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestDaggerheartHandlersUseSharedDomainWriteHelper(t *testing.T) {
	var violations []string
	for _, filename := range []string{"actions.go", "adversaries.go"} {
		sourcePath := localSourcePath(t, filename)
		domainAliases, err := findDomainStoreAliases(sourcePath)
		if err != nil {
			t.Fatalf("scan aliases in %s: %v", filename, err)
		}
		lines, err := findCallLines(sourcePath, func(callPath string, _ *ast.CallExpr) bool {
			if callPath == "s.stores.Domain.Execute" {
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

func TestDaggerheartHandlersDoNotInlineApplyEvents(t *testing.T) {
	var violations []string
	for _, filename := range []string{"actions.go", "adversaries.go"} {
		sourcePath := localSourcePath(t, filename)
		lines, err := findCallLines(sourcePath, func(callPath string, _ *ast.CallExpr) bool {
			return strings.HasSuffix(callPath, ".Apply")
		})
		if err != nil {
			t.Fatalf("scan %s: %v", filename, err)
		}
		violations = append(violations, linesWithFile(filename, lines)...)
	}

	assertNoViolations(t, "inline Apply(ctx, evt) calls found in Daggerheart handlers", violations)
}

func TestDaggerheartHandlersNoDirectStorageMutationBypass(t *testing.T) {
	tests := []struct {
		name               string
		filename           string
		disallowedCalls    []string
		disallowedLiterals []string
	}{
		{
			name:     "actions",
			filename: "actions.go",
			disallowedCalls: []string{
				"s.stores.Event.AppendEvent",
			},
			disallowedLiterals: []string{
				"action.outcome_rejected",
				"story.note_added",
			},
		},
		{
			name:     "adversaries",
			filename: "adversaries.go",
			disallowedCalls: []string{
				"s.stores.Event.AppendEvent",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sourcePath := localSourcePath(t, tc.filename)

			callLines, err := findCallLines(sourcePath, func(callPath string, _ *ast.CallExpr) bool {
				for _, disallowed := range tc.disallowedCalls {
					if callPath == disallowed {
						return true
					}
				}
				return hasDisallowedStoreMutationCall(callPath)
			})
			if err != nil {
				t.Fatalf("scan calls in %s: %v", tc.filename, err)
			}

			literalLines, err := findStringLiteralLines(sourcePath, tc.disallowedLiterals)
			if err != nil {
				t.Fatalf("scan literals in %s: %v", tc.filename, err)
			}

			var violations []string
			violations = append(violations, linesWithFile(tc.filename, callLines)...)
			violations = append(violations, linesWithFile(tc.filename, literalLines)...)
			assertNoViolations(t, fmt.Sprintf("%s contains write-path bypass patterns", tc.filename), violations)
		})
	}
}

func localSourcePath(t *testing.T, filename string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}
	return filepath.Join(filepath.Dir(thisFile), filename)
}

func findCallLines(path string, disallowed func(callPath string, call *ast.CallExpr) bool) ([]int, error) {
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
		callPath := selectorPath(call.Fun)
		if callPath == "" || !disallowed(callPath, call) {
			return true
		}
		lines = append(lines, fset.Position(call.Lparen).Line)
		return true
	})
	return lines, nil
}

func findStringLiteralLines(path string, values []string) ([]int, error) {
	if len(values) == 0 {
		return nil, nil
	}
	valueSet := make(map[string]struct{}, len(values))
	for _, value := range values {
		valueSet[value] = struct{}{}
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	lines := make([]int, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		if _, exists := valueSet[value]; exists {
			lines = append(lines, fset.Position(lit.ValuePos).Line)
		}
		return true
	})
	return lines, nil
}

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
	return selectorPath(expr) == "s.stores.Domain"
}

func selectorPath(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		prefix := selectorPath(typed.X)
		if prefix == "" {
			return typed.Sel.Name
		}
		return prefix + "." + typed.Sel.Name
	case *ast.Ident:
		return typed.Name
	case *ast.ParenExpr:
		return selectorPath(typed.X)
	case *ast.StarExpr:
		return selectorPath(typed.X)
	default:
		return ""
	}
}

func hasDisallowedStoreMutationCall(callPath string) bool {
	return strings.Contains(callPath, ".PutDaggerheart") ||
		strings.Contains(callPath, ".UpdateDaggerheart") ||
		strings.Contains(callPath, ".DeleteDaggerheart")
}

func linesWithFile(filename string, lines []int) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, fmt.Sprintf("%s:%d", filename, line))
	}
	return out
}

func assertNoViolations(t *testing.T, message string, violations []string) {
	t.Helper()
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("%s:\n%s", message, strings.Join(violations, "\n"))
}
