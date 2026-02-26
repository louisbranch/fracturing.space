package modules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestFeatureModulesDoNotImportSiblingModules(t *testing.T) {
	t.Parallel()

	entries, err := filepath.Glob(filepath.Join("*", "*.go"))
	if err != nil {
		t.Fatalf("glob module files: %v", err)
	}
	fset := token.NewFileSet()
	for _, file := range entries {
		parsed, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports for %s: %v", file, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if strings.Contains(path, "/internal/services/web/modules/") {
				t.Fatalf("file %s imports sibling module path %q", file, path)
			}
		}
	}
}

func TestRoutePrefixesRemainUniqueConstants(t *testing.T) {
	t.Parallel()

	prefixes := []string{
		routepath.AuthPrefix,
		routepath.DiscoverPrefix,
		routepath.UserProfilePrefix,
		routepath.DashboardPrefix,
		routepath.CampaignsPrefix,
		routepath.Notifications,
		routepath.ProfilePrefix,
		routepath.SettingsPrefix,
	}
	seen := map[string]struct{}{}
	for _, prefix := range prefixes {
		if _, ok := seen[prefix]; ok {
			t.Fatalf("duplicate route prefix constant %q", prefix)
		}
		seen[prefix] = struct{}{}
	}
}

func TestFeatureModulesFollowTemplate(t *testing.T) {
	t.Parallel()

	areas := []string{
		"public",
		"dashboard",
		"campaigns",
		"discovery",
		"publicprofile",
		"notifications",
		"profile",
		"settings",
	}
	requiredFiles := []string{"module.go", "routes.go", "routes_test.go", "handlers.go", "service.go"}
	for _, area := range areas {
		for _, file := range requiredFiles {
			path := filepath.Join(area, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("module %q missing required file %q: %v", area, file, err)
			}
		}
	}
}

func TestCampaignsAndSettingsMountDoNotReadGatewayClientsFromDependencies(t *testing.T) {
	t.Parallel()

	// TODO(web-guardrails): generalize this guard so new modules cannot silently bypass composition-owned gateway wiring rules.
	assertMountDoesNotReadDependencyFields(t, filepath.Join("campaigns", "module.go"), map[string]struct{}{
		"CampaignClient":    {},
		"ParticipantClient": {},
		"CharacterClient":   {},
		"AssetBaseURL":      {},
	})
	assertMountDoesNotReadDependencyFields(t, filepath.Join("settings", "module.go"), map[string]struct{}{
		"SocialClient":     {},
		"AccountClient":    {},
		"CredentialClient": {},
	})
}

func assertMountDoesNotReadDependencyFields(t *testing.T, moduleFile string, forbidden map[string]struct{}) {
	t.Helper()

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, moduleFile, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse module file %s: %v", moduleFile, err)
	}

	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "Mount" || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "deps" {
				return true
			}
			if _, exists := forbidden[sel.Sel.Name]; exists {
				t.Fatalf("%s Mount reads deps.%s; wire gateways in composition instead", moduleFile, sel.Sel.Name)
			}
			return true
		})
		return
	}

	t.Fatalf("module file %s missing Mount function", moduleFile)
}
