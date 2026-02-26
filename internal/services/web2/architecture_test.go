package web2

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestWeb2ProductionCodeDoesNotImportLegacyWebPackages(t *testing.T) {
	t.Parallel()

	const legacyWebPrefix = "github.com/louisbranch/fracturing.space/internal/services/web/"

	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		parsed, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse imports for %s: %w", path, err)
		}
		for _, imp := range parsed.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			if strings.HasPrefix(importPath, legacyWebPrefix) {
				return fmt.Errorf("file %s imports legacy web package %q", path, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
