package modules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestFeatureModulesDoNotImportSiblingModules(t *testing.T) {
	t.Parallel()

	for _, mod := range discoverModules(t) {
		files := moduleGoFiles(t, mod, false)
		if len(files) == 0 {
			continue
		}
		for _, file := range files {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse imports for %s: %v", file, err)
			}
			assertNoSiblingModuleImport(t, file, parsed.Imports)
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

	protectedModulesWithGateway := map[string]struct{}{
		"campaigns":     {},
		"dashboard":     {},
		"notifications": {},
		"settings":      {},
	}
	for _, mod := range discoverModules(t) {
		for _, file := range []string{"module.go", "routes.go", "routes_test.go"} {
			path := filepath.Join(mod, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("module %q missing required file %q: %v", mod, file, err)
			}
		}
		if len(moduleHandlerFiles(mod)) == 0 {
			t.Fatalf("module %q has no handler files matching *handlers*.go", mod)
		}
		if len(moduleServiceFiles(mod)) == 0 {
			t.Fatalf("module %q has no service files matching *service*.go", mod)
		}
		if _, ok := protectedModulesWithGateway[mod]; ok {
			unavailFile := "gateway_unavailable.go"
			path := filepath.Join(mod, unavailFile)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("protected module %q missing required file %q: %v", mod, unavailFile, err)
			}
		}
	}
}

func TestModulesMountDoNotReadGatewayClientsFromDependencies(t *testing.T) {
	t.Parallel()

	assertMountDoNotReadDependencyFields(t, filepath.Join("campaigns", "module.go"), map[string]struct{}{
		"CampaignClient":    {},
		"ParticipantClient": {},
		"CharacterClient":   {},
		"AssetBaseURL":      {},
	})
	assertMountDoNotReadDependencyFields(t, filepath.Join("settings", "module.go"), map[string]struct{}{
		"SocialClient":     {},
		"AccountClient":    {},
		"CredentialClient": {},
	})
	assertMountDoNotReadDependencyFields(t, filepath.Join("notifications", "module.go"), map[string]struct{}{
		"NotificationClient": {},
	})
	assertMountDoNotReadGRPCClient(t, filepath.Join("public", "module.go"))
	assertMountDoNotReadGRPCClient(t, filepath.Join("profile", "module.go"))
}

func TestProtectedModuleHandlersDoNotBypassBaseResolverMethods(t *testing.T) {
	t.Parallel()

	// Protected module handlers embed modulehandler.Base whose resolver fields
	// are unexported. This guard detects direct calls to webctx.WithResolvedUserID
	// or webi18n.ResolveTag in handler files, which would bypass the designed
	// Base methods (RequestContextAndUserID, RequestLocaleTag, PageLocalizer, etc.).
	forbiddenImports := map[string]struct{}{
		"github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx": {},
	}
	protectedModules := []string{"campaigns", "dashboard", "notifications", "settings"}
	for _, mod := range protectedModules {
		for _, file := range moduleHandlerFiles(mod) {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse handler file %s: %v", file, err)
			}
			for _, imp := range parsed.Imports {
				path := strings.Trim(imp.Path.Value, "\"")
				if _, exists := forbiddenImports[path]; exists {
					t.Errorf("%s imports %s; use modulehandler.Base methods instead of raw webctx calls", file, path)
				}
			}
		}
	}
}

func TestProtectedModuleServicesDoNotImportWebContext(t *testing.T) {
	t.Parallel()

	// Service files should not import the webctx package. User ID validation
	// belongs as a local helper, not as a dependency on web-platform plumbing.
	forbidden := "github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx"
	protectedModules := []string{"campaigns", "dashboard", "notifications", "settings"}
	for _, mod := range protectedModules {
		for _, file := range moduleServiceFiles(mod) {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse service file %s: %v", file, err)
			}
			for _, imp := range parsed.Imports {
				path := strings.Trim(imp.Path.Value, "\"")
				if path == forbidden {
					t.Errorf("%s imports %s; use a local requireUserID helper instead", file, path)
				}
			}
		}
	}
}

func TestProtectedModuleHandlersDoNotImportProtoPackages(t *testing.T) {
	t.Parallel()

	// Handler files should use domain types, not proto types.
	// Proto → domain mapping belongs in gateway files.
	forbiddenPrefix := "github.com/louisbranch/fracturing.space/api/gen/go/"
	protectedModules := []string{"campaigns", "dashboard", "notifications", "settings"}
	for _, mod := range protectedModules {
		for _, file := range moduleHandlerFiles(mod) {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse handler file %s: %v", file, err)
			}
			for _, imp := range parsed.Imports {
				path := strings.Trim(imp.Path.Value, "\"")
				if strings.HasPrefix(path, forbiddenPrefix) {
					t.Errorf("%s imports proto package %s; use domain types and map at the gateway boundary", file, path)
				}
			}
		}
	}
}

// assertMountDoesNotReadGRPCClient verifies that Mount() does not call any
// function whose name contains "GRPC" or "grpc" — i.e. the module receives
// a pre-built gateway and does not construct one from a raw client.
func assertMountDoNotReadGRPCClient(t *testing.T, moduleFile string) {
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
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if strings.Contains(strings.ToLower(ident.Name), "grpc") {
				t.Fatalf("%s Mount calls %s; inject pre-built gateways from registry instead", moduleFile, ident.Name)
			}
			return true
		})
		return
	}

	t.Fatalf("module file %s missing Mount function", moduleFile)
}

func assertMountDoNotReadDependencyFields(t *testing.T, moduleFile string, forbidden map[string]struct{}) {
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

func discoverModules(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read modules directory: %v", err)
	}
	modules := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		files, err := filepath.Glob(filepath.Join(entry.Name(), "*.go"))
		if err != nil {
			t.Fatalf("glob module files for %q: %v", entry.Name(), err)
		}
		if len(files) == 0 {
			continue
		}
		if entry.Name() == "testdata" {
			continue
		}
		modules = append(modules, entry.Name())
	}
	sort.Strings(modules)
	return modules
}

func moduleGoFiles(t *testing.T, module string, includeTests bool) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(module, "*.go"))
	if err != nil {
		t.Fatalf("glob go files for module %q: %v", module, err)
	}
	filtered := files[:0]
	for _, file := range files {
		if includeTests || !strings.HasSuffix(file, "_test.go") {
			filtered = append(filtered, file)
		}
	}
	sort.Strings(filtered)
	return filtered
}

func moduleHandlerFiles(module string) []string {
	files, _ := filepath.Glob(filepath.Join(module, "*handlers*.go"))
	candidates := files[:0]
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		candidates = append(candidates, file)
	}
	sort.Strings(candidates)
	return candidates
}

func moduleServiceFiles(module string) []string {
	files, _ := filepath.Glob(filepath.Join(module, "*service*.go"))
	candidates := files[:0]
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		candidates = append(candidates, file)
	}
	sort.Strings(candidates)
	return candidates
}

func assertNoSiblingModuleImport(t *testing.T, file string, imports []*ast.ImportSpec) {
	t.Helper()

	for _, imp := range imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if strings.Contains(path, "/internal/services/web/modules/") {
			t.Fatalf("file %s imports sibling module path %q", file, path)
		}
	}
}
