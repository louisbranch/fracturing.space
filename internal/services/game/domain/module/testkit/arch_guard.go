// Package testkit provides reusable test helpers for validating system module
// conformance and write-path architecture guards.
//
// The arch guard helpers (ScanCallViolations, ScanLiteralViolations,
// ScanHandlerDir) use Go AST parsing to detect bypass patterns in handler
// code — direct storage mutations, inline event appends, and forbidden string
// literals — so each game system can enforce the same rules without duplicating
// the scanning logic.
package testkit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ScanCallViolations parses a Go source file and returns line numbers where
// a function call matches the disallowed predicate. The predicate receives the
// dot-separated selector path of each call expression (e.g. "s.stores.Domain.Execute").
func ScanCallViolations(path string, disallowed func(callPath string) bool) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var lines []int
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		callPath := selectorPath(call.Fun)
		if callPath != "" && disallowed(callPath) {
			lines = append(lines, fset.Position(call.Lparen).Line)
		}
		return true
	})
	return lines, nil
}

// ScanLiteralViolations parses a Go source file and returns line numbers
// where a string literal matches one of the forbidden values.
func ScanLiteralViolations(path string, forbidden []string) ([]int, error) {
	if len(forbidden) == 0 {
		return nil, nil
	}
	valueSet := make(map[string]struct{}, len(forbidden))
	for _, v := range forbidden {
		valueSet[v] = struct{}{}
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var lines []int
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

// ScanHandlerDir returns sorted filenames of non-test Go files in dir.
func ScanHandlerDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files, nil
}

// WritePathPolicy configures which patterns the write-path architecture
// guard should flag as violations. Each system provides its own policy
// so the scanning logic is reusable across game systems.
type WritePathPolicy struct {
	// HandlerDir is the directory containing handler source files.
	HandlerDir string
	// StoreMutationSubstrings lists dot-path substrings that indicate
	// direct store mutation bypassing the event pipeline
	// (e.g. ".PutDaggerheart", ".UpdateDaggerheart").
	StoreMutationSubstrings []string
	// LiteralPolicies maps filenames to forbidden string literals.
	// Literals are checked only in the specified files.
	LiteralPolicies map[string][]string
}

// ValidateWritePathArchitecture runs the three standard write-path guards
// against handler files in the policy directory:
//
//  1. No inline .Apply calls (events go through journal, not applied directly)
//  2. No direct storage mutation (bypass patterns from StoreMutationSubstrings)
//  3. No forbidden string literals (per-file LiteralPolicies)
//
// The Domain.Execute check is intentionally excluded because the shared-helper
// pattern (and its alias detection) is system-specific. Systems that need it
// can add a dedicated test using ScanCallViolations directly.
func ValidateWritePathArchitecture(t testing.TB, policy WritePathPolicy) {
	t.Helper()
	files, err := ScanHandlerDir(policy.HandlerDir)
	if err != nil {
		t.Fatalf("load architecture scan files: %v", err)
	}

	for _, filename := range files {
		sourcePath := filepath.Join(policy.HandlerDir, filename)

		// Guard 1: No inline .Apply calls.
		applyLines, err := ScanCallViolations(sourcePath, func(callPath string) bool {
			return strings.HasSuffix(callPath, ".Apply")
		})
		if err != nil {
			t.Fatalf("scan .Apply in %s: %v", filename, err)
		}
		for _, line := range applyLines {
			t.Errorf("%s:%d: inline .Apply call bypasses event journal", filename, line)
		}

		// Guard 2: No direct storage mutation or event append.
		mutationLines, err := ScanCallViolations(sourcePath, func(callPath string) bool {
			if callPath == "s.stores.Event.AppendEvent" {
				return true
			}
			for _, substr := range policy.StoreMutationSubstrings {
				if strings.Contains(callPath, substr) {
					return true
				}
			}
			return false
		})
		if err != nil {
			t.Fatalf("scan mutation in %s: %v", filename, err)
		}
		for _, line := range mutationLines {
			t.Errorf("%s:%d: direct storage mutation bypasses event pipeline", filename, line)
		}

		// Guard 3: Forbidden string literals.
		if forbidden, ok := policy.LiteralPolicies[filename]; ok {
			literalLines, err := ScanLiteralViolations(sourcePath, forbidden)
			if err != nil {
				t.Fatalf("scan literals in %s: %v", filename, err)
			}
			for _, line := range literalLines {
				t.Errorf("%s:%d: forbidden string literal", filename, line)
			}
		}
	}
}

// selectorPath resolves the dot-separated path of a selector expression
// (e.g. s.stores.Domain.Execute → "s.stores.Domain.Execute").
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
