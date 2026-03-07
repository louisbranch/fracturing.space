package game

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestPayloadPointerFieldsRequireOmitempty verifies that pointer fields in
// payload structs include the `omitempty` JSON tag. Pointer fields represent
// optional-with-meaning semantics; without `omitempty`, serialized events
// contain explicit `null` values that create ambiguity between "absent" and
// "explicitly null" during replay. This catches replay-breaking additions
// before they enter the event journal.
//
// See docs/architecture/policy/event-payload-change-policy.md.
func TestPayloadPointerFieldsRequireOmitempty(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	domainRoot := filepath.Join(repoRoot, "internal", "services", "game", "domain")

	var violations []string
	walkErr := filepath.WalkDir(domainRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "payload.go" {
			return nil
		}
		fileViolations, err := findPointerFieldsMissingOmitempty(path)
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(repoRoot, path)
		for _, v := range fileViolations {
			violations = append(violations, fmt.Sprintf("%s:%d %s", filepath.ToSlash(relPath), v.line, v.field))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan payload files: %v", walkErr)
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Errorf(
		"pointer fields in payload structs must include `omitempty` in their JSON tag "+
			"to avoid null vs absent ambiguity during event replay.\n"+
			"See docs/architecture/policy/event-payload-change-policy.md.\n\n"+
			"Violations:\n%s",
		strings.Join(violations, "\n"),
	)
}

type pointerFieldViolation struct {
	line  int
	field string
}

func findPointerFieldsMissingOmitempty(path string) ([]pointerFieldViolation, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var violations []pointerFieldViolation
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if !strings.HasSuffix(typeSpec.Name.Name, "Payload") {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structType.Fields.List {
				if !isPointerType(field.Type) {
					continue
				}
				if field.Tag == nil {
					for _, name := range field.Names {
						violations = append(violations, pointerFieldViolation{
							line:  fset.Position(name.Pos()).Line,
							field: fmt.Sprintf("%s.%s: missing json tag", typeSpec.Name.Name, name.Name),
						})
					}
					continue
				}
				jsonTag := extractJSONTag(field.Tag.Value)
				if jsonTag == "" || jsonTag == "-" {
					continue
				}
				if !strings.Contains(jsonTag, "omitempty") {
					for _, name := range field.Names {
						violations = append(violations, pointerFieldViolation{
							line:  fset.Position(name.Pos()).Line,
							field: fmt.Sprintf("%s.%s: json tag %q lacks omitempty", typeSpec.Name.Name, name.Name, jsonTag),
						})
					}
				}
			}
		}
	}
	return violations, nil
}

func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func extractJSONTag(rawTag string) string {
	// rawTag includes backticks: `json:"name,omitempty"`
	tag := strings.Trim(rawTag, "`")
	for _, part := range strings.Fields(tag) {
		if strings.HasPrefix(part, "json:") {
			return strings.Trim(strings.TrimPrefix(part, "json:"), "\"")
		}
	}
	return ""
}
