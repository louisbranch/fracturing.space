package modules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
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
		routepath.InvitePrefix,
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

	const (
		archetypeTransportOnly    = "transport_only"
		archetypeTransportLayered = "transport_with_app_gateway"
	)
	moduleArchetypes := map[string]string{
		"campaigns":     archetypeTransportLayered,
		"dashboard":     archetypeTransportLayered,
		"discovery":     archetypeTransportLayered,
		"invite":        archetypeTransportLayered,
		"notifications": archetypeTransportLayered,
		"profile":       archetypeTransportLayered,
		"publicauth":    archetypeTransportLayered,
		"settings":      archetypeTransportLayered,
	}
	protectedModuleGatewayUnavailableFiles := map[string]string{
		"campaigns":     filepath.Join("app", "unavailable_gateway.go"),
		"dashboard":     filepath.Join("app", "unavailable_gateway.go"),
		"notifications": filepath.Join("app", "unavailable_gateway.go"),
		"settings":      filepath.Join("app", "unavailable_gateway.go"),
	}
	moduleAppBoundaryFiles := map[string][]string{
		"publicauth": {
			filepath.Join("app", "doc.go"),
			filepath.Join("app", "service_page.go"),
			filepath.Join("app", "service_session.go"),
			filepath.Join("app", "service_passkey.go"),
			filepath.Join("app", "service_recovery.go"),
			filepath.Join("gateway", "doc.go"),
		},
	}
	for _, mod := range discoverModules(t) {
		archetype, ok := moduleArchetypes[mod]
		if !ok {
			t.Fatalf("module %q missing archetype classification", mod)
		}
		for _, file := range []string{"module.go", "routes.go", "routes_test.go"} {
			path := filepath.Join(mod, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("module %q missing required file %q: %v", mod, file, err)
			}
		}
		if len(moduleHandlerFiles(t, mod)) == 0 {
			t.Fatalf("module %q has no handler files matching *handlers*.go", mod)
		}
		switch archetype {
		case archetypeTransportOnly:
			// transport-only modules intentionally keep orchestration minimal and
			// do not require root/service or app/gateway subpackages.
		case archetypeTransportLayered:
			requiredFiles := []string{
				filepath.Join("app", "doc.go"),
				filepath.Join("app", "service.go"),
				filepath.Join("gateway", "doc.go"),
			}
			if customFiles, ok := moduleAppBoundaryFiles[mod]; ok {
				requiredFiles = customFiles
			}
			for _, file := range requiredFiles {
				path := filepath.Join(mod, file)
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("module %q missing layered boundary file %q: %v", mod, file, err)
				}
			}
			if len(goFilesUnder(t, filepath.Join(mod, "gateway"), false)) == 0 {
				t.Fatalf("module %q has no gateway implementation files", mod)
			}
		default:
			t.Fatalf("module %q has unknown archetype %q", mod, archetype)
		}
		if unavailFile, ok := protectedModuleGatewayUnavailableFiles[mod]; ok {
			path := filepath.Join(mod, unavailFile)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("protected module %q missing required file %q: %v", mod, unavailFile, err)
			}
		}
	}
}

func TestSelectedModulesKeepContributorOwnedTransportSplits(t *testing.T) {
	t.Parallel()

	requiredFilesByModule := map[string][]string{
		"settings": {
			"handlers_profile.go",
			"handlers_locale.go",
			"handlers_ai_keys.go",
			"handlers_ai_agents.go",
			"handlers_shell.go",
			"routes_account.go",
			"routes_ai.go",
		},
	}

	for mod, files := range requiredFilesByModule {
		for _, file := range files {
			path := filepath.Join(mod, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("module %q missing contributor-owned transport file %q: %v", mod, file, err)
			}
		}
	}
}

func TestSelectedModulesKeepAreaOwnedCompositionEntrypoints(t *testing.T) {
	t.Parallel()

	for _, mod := range []string{
		"campaigns",
		"dashboard",
		"discovery",
		"invite",
		"notifications",
		"profile",
		"publicauth",
		"settings",
	} {
		path := filepath.Join(mod, "composition.go")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("module %q missing area-owned composition entrypoint %q: %v", mod, path, err)
		}
	}
}

func TestSelectedModulesKeepContributorOwnedAppAndGatewaySplits(t *testing.T) {
	t.Parallel()

	requiredFilesByModule := map[string][]string{
		"settings": {
			filepath.Join("app", "service_account.go"),
			filepath.Join("app", "service_ai.go"),
			filepath.Join("app", "unavailable_account.go"),
			filepath.Join("app", "unavailable_ai.go"),
			filepath.Join("gateway", "grpc_account.go"),
			filepath.Join("gateway", "grpc_ai.go"),
		},
	}

	for mod, files := range requiredFilesByModule {
		for _, file := range files {
			path := filepath.Join(mod, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("module %q missing contributor-owned app/gateway file %q: %v", mod, file, err)
			}
		}
	}
}

func TestWebContributorDocsReferenceCoverageEntrypoints(t *testing.T) {
	t.Parallel()

	docRefs := map[string][]string{
		filepath.Join("..", "..", "..", "..", "docs", "architecture", "platform", "web-contributor-map.md"): {
			"internal/services/web/server_test.go",
			"internal/services/web/server_locale_test.go",
			"internal/services/web/server_viewer_test.go",
			"internal/services/web/server_static_test.go",
			"internal/services/web/server_test_harness_defaults_test.go",
			"internal/services/web/server_test_harness_helpers_test.go",
			"internal/cmd/web/web_test.go",
			"docs/architecture/platform/web-testing-map.md",
		},
		filepath.Join("..", "..", "..", "..", "docs", "architecture", "platform", "web-testing-map.md"): {
			"internal/services/web/server_test.go",
			"internal/services/web/server_locale_test.go",
			"internal/services/web/server_viewer_test.go",
			"internal/services/web/server_static_test.go",
			"internal/services/web/server_test_harness_defaults_test.go",
			"internal/services/web/server_test_harness_helpers_test.go",
			"internal/cmd/web/web_test.go",
			"internal/services/web/modules/architecture_test.go",
			"internal/services/web/modules/boundary_guardrails_test.go",
			"make test",
			"make web-architecture-check",
			"make smoke",
			"make check",
		},
		filepath.Join("..", "..", "..", "..", "docs", "guides", "web-module-playbook.md"): {
			"web-testing-map.md",
			"internal/services/web/modules/architecture_test.go",
			"internal/services/web/modules/boundary_guardrails_test.go",
			"routes_test.go",
			"handlers*_test.go",
		},
		filepath.Join("..", "..", "..", "..", "CONTRIBUTING.md"): {
			"docs/architecture/platform/web-testing-map.md",
		},
		filepath.Join("..", "..", "..", "..", "docs", "running", "verification.md"): {
			"web-testing-map.md",
		},
	}

	for docPath, refs := range docRefs {
		content, err := os.ReadFile(docPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", docPath, err)
		}
		body := string(content)
		for _, ref := range refs {
			if !strings.Contains(body, ref) {
				t.Fatalf("%s does not mention %q", docPath, ref)
			}
			if strings.HasPrefix(ref, "internal/") || strings.HasPrefix(ref, "docs/") || ref == "CONTRIBUTING.md" {
				repoPath := filepath.Join("..", "..", "..", "..", ref)
				if _, err := os.Stat(repoPath); err != nil {
					t.Fatalf("doc reference %q from %s is not present in repo: %v", ref, docPath, err)
				}
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
		"SettingsSocialClient": {},
		"AccountClient":        {},
		"CredentialClient":     {},
		"AgentClient":          {},
	})
	assertMountDoNotReadDependencyFields(t, filepath.Join("notifications", "module.go"), map[string]struct{}{
		"NotificationClient": {},
	})
	assertMountDoNotReadGRPCClient(t, filepath.Join("publicauth", "module.go"))
	assertMountDoNotReadGRPCClient(t, filepath.Join("profile", "module.go"))
}

func TestProtectedModuleHandlersDoNotBypassBaseResolverMethods(t *testing.T) {
	t.Parallel()

	// Protected module handlers embed modulehandler.Base, which now embeds the
	// shared principal request-state seam while keeping user-id helpers local. This
	// guard detects direct calls to webctx.WithResolvedUserID that would bypass
	// the designed Base methods (RequestContextAndUserID, RequestLocaleTag,
	// PageLocalizer, etc.).
	forbiddenImports := map[string]struct{}{
		"github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx": {},
	}
	protectedModules := []string{"campaigns", "dashboard", "notifications", "settings"}
	for _, mod := range protectedModules {
		for _, file := range moduleHandlerFiles(t, mod) {
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
		for _, file := range moduleServiceFiles(t, mod) {
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

func TestProfileAppDoesNotImportAvatarURLFormatting(t *testing.T) {
	t.Parallel()

	forbidden := "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	for _, file := range goFilesUnder(t, filepath.Join("profile", "app"), false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse app file %s: %v", file, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if path == forbidden {
				t.Errorf("%s imports %s; avatar URL formatting belongs in transport/view mapping, not profile/app", file, path)
			}
		}
	}
}

func TestNotificationsModuleDoesNotImportNotificationsServicePackages(t *testing.T) {
	t.Parallel()

	const forbiddenPrefix = "github.com/louisbranch/fracturing.space/internal/services/notifications/"
	for _, file := range goFilesUnder(t, "notifications", false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse notifications file %s: %v", file, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if strings.HasPrefix(path, forbiddenPrefix) {
				t.Errorf("%s imports %s; notifications web copy/rendering must stay area-owned inside internal/services/web/modules/notifications", file, path)
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
		for _, file := range moduleHandlerFiles(t, mod) {
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

func TestSelectedModuleHandlersDoNotReadRawPathValues(t *testing.T) {
	t.Parallel()

	modulesUsingSharedRouteParamHelper := []string{"campaigns", "notifications", "settings", "profile"}
	for _, mod := range modulesUsingSharedRouteParamHelper {
		for _, file := range moduleHandlerFiles(t, mod) {
			parsed := parseFile(t, file)
			ast.Inspect(parsed, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil || sel.Sel.Name != "PathValue" {
					return true
				}
				t.Errorf("%s calls raw PathValue; use platform/routeparam helpers instead", file)
				return true
			})
		}
	}
}

func TestCampaignAppPackageDoesNotImportTransportOrTemplates(t *testing.T) {
	t.Parallel()

	files := goFilesUnder(t, filepath.Join("campaigns", "app"), false)
	forbidden := map[string]string{
		"net/http": "app package should stay transport-free; parse forms in handlers/workflows",
		"github.com/louisbranch/fracturing.space/internal/services/web/templates": "app package should return domain models, not template views",
	}

	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse app file %s: %v", file, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if why, exists := forbidden[path]; exists {
				t.Errorf("%s imports %s: %s", file, path, why)
			}
		}
	}
}

func TestCampaignWorkflowPackagesDoNotImportRootCampaignsPackage(t *testing.T) {
	t.Parallel()

	const rootCampaignsImport = "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	for _, file := range goFilesUnder(t, filepath.Join("campaigns", "workflow"), false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse workflow file %s: %v", file, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if path == rootCampaignsImport {
				t.Errorf("%s imports %s; use campaigns/app contracts to avoid parent-package coupling", file, path)
			}
		}
	}
}

func TestCampaignWorkflowPackagesDoNotImportCampaignRender(t *testing.T) {
	t.Parallel()

	const renderImport = "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	for _, file := range goFilesUnder(t, filepath.Join("campaigns", "workflow"), false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse workflow file %s: %v", file, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if path == renderImport {
				t.Errorf("%s imports %s; workflow-owned models should adapt at the render seam instead", file, path)
			}
		}
	}
}

func TestGatewayPackagesDoNotImportTemplates(t *testing.T) {
	t.Parallel()

	const templatesImport = "github.com/louisbranch/fracturing.space/internal/services/web/templates"

	for _, mod := range discoverModules(t) {
		for _, file := range moduleGoFiles(t, mod, false) {
			if !isGatewayFile(file) {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse gateway file %s: %v", file, err)
			}
			for _, imp := range parsed.Imports {
				path := strings.Trim(imp.Path.Value, "\"")
				if path == templatesImport {
					t.Errorf("%s imports %s; gateway code should map transport/domain contracts only", file, path)
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

	return goFilesUnder(t, module, includeTests)
}

func moduleHandlerFiles(t *testing.T, module string) []string {
	t.Helper()

	files := moduleGoFiles(t, module, false)
	candidates := make([]string, 0, len(files))
	for _, file := range files {
		if strings.Contains(filepath.Base(file), "handlers") {
			candidates = append(candidates, file)
		}
	}
	sort.Strings(candidates)
	return candidates
}

func moduleServiceFiles(t *testing.T, module string) []string {
	t.Helper()

	files := moduleGoFiles(t, module, false)
	candidates := make([]string, 0, len(files))
	for _, file := range files {
		if strings.Contains(filepath.Base(file), "service") {
			candidates = append(candidates, file)
		}
	}
	sort.Strings(candidates)
	return candidates
}

func assertNoSiblingModuleImport(t *testing.T, file string, imports []*ast.ImportSpec) {
	t.Helper()

	moduleName := strings.Split(filepath.ToSlash(file), "/")[0]
	const modulesPath = "/internal/services/web/modules/"

	for _, imp := range imports {
		path := strings.Trim(imp.Path.Value, "\"")
		index := strings.Index(path, modulesPath)
		if index < 0 {
			continue
		}
		modulePath := path[index+len(modulesPath):]
		importedModule := strings.Split(modulePath, "/")[0]
		if importedModule != moduleName {
			t.Fatalf("file %s imports sibling module path %q", file, path)
		}
	}
}

func goFilesUnder(t *testing.T, root string, includeTests bool) []string {
	t.Helper()

	files := make([]string, 0, 16)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk go files under %q: %v", root, err)
	}
	sort.Strings(files)
	return files
}

func isGatewayFile(path string) bool {
	path = filepath.ToSlash(path)
	if strings.Contains(path, "/gateway/") {
		return true
	}
	return strings.Contains(filepath.Base(path), "gateway")
}
