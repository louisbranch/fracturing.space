//go:build integration
// +build integration

package integration

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLegacyCampaignEventPackageImportsAreAllowlisted(t *testing.T) {
	const legacyImportPath = "github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"

	root := integrationRepoRoot(t)
	allowlist := legacyCampaignEventImportAllowlist()
	var violations []string

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if importPath != legacyImportPath {
				continue
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if _, ok := allowlist[rel]; ok {
				continue
			}
			violations = append(violations, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan legacy event imports: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("legacy campaign event package imports must be allowlisted:\n- %s", strings.Join(violations, "\n- "))
	}
}

func legacyCampaignEventImportAllowlist() map[string]struct{} {
	return map[string]struct{}{}
}
