package modules

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

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

const (
	moduleImportPrefix       = "github.com/louisbranch/fracturing.space/internal/services/admin/modules/"
	legacyAreaImportPrefix   = "github.com/louisbranch/fracturing.space/internal/services/admin/module/"
	adminLegacyModuleDirName = "module"
	rootAdapterFileName      = "module_adapters.go"
)

var requiredModuleFiles = []string{
	"module.go",
	"handlers.go",
	"routes.go",
	"module_test.go",
}

func TestAreaModulesDoNotImportSiblingModules(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	for _, name := range discoverAreaModules(t, root) {
		files := moduleGoFiles(t, filepath.Join(root, name))
		for _, file := range files {
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse imports for %s: %v", file, err)
			}
			for _, imp := range parsed.Imports {
				path := strings.Trim(imp.Path.Value, "\"")
				if strings.HasPrefix(path, legacyAreaImportPrefix) {
					t.Fatalf("%s imports legacy area package %q; legacy module/<area> paths are removed", file, path)
				}
				if !strings.HasPrefix(path, moduleImportPrefix) {
					continue
				}
				if strings.HasPrefix(path, moduleImportPrefix+name) {
					continue
				}
				t.Fatalf("%s imports sibling area module %q; keep area ownership local", file, path)
			}
		}
	}
}

func TestRegistryModuleMountPrefixesAreUniqueAndAppScoped(t *testing.T) {
	t.Parallel()

	built := NewRegistry().Build(BuildInput{})
	seen := map[string]struct{}{}
	for _, mod := range built.Modules {
		mount, err := mod.Mount()
		if err != nil {
			t.Fatalf("mount module %q: %v", mod.ID(), err)
		}
		prefix := strings.TrimSpace(mount.Prefix)
		if prefix == "" {
			t.Fatalf("module %q has empty prefix", mod.ID())
		}
		if !strings.HasPrefix(prefix, routepath.AppPrefix) {
			t.Fatalf("module %q prefix = %q, want %q prefix", mod.ID(), prefix, routepath.AppPrefix)
		}
		if _, ok := seen[prefix]; ok {
			t.Fatalf("duplicate module prefix %q", prefix)
		}
		seen[prefix] = struct{}{}
	}
}

func TestLegacyAreaModuleDirectoriesAreRemoved(t *testing.T) {
	t.Parallel()

	adminRoot := filepath.Dir(moduleRoot(t))
	legacy := []string{
		"campaigns",
		"catalog",
		"dashboard",
		"icons",
		"scenarios",
		"sharedpath",
		"systems",
		"users",
	}

	for _, dir := range legacy {
		path := filepath.Join(adminRoot, adminLegacyModuleDirName, dir)
		_, err := os.Stat(path)
		if err == nil {
			t.Fatalf("legacy directory %q exists; remove old module/<area> package", path)
		}
		if !os.IsNotExist(err) {
			t.Fatalf("stat %q: %v", path, err)
		}
	}
}

func TestRootAdminDoesNotOwnAreaHandleMethods(t *testing.T) {
	t.Parallel()

	adminRoot := filepath.Dir(moduleRoot(t))
	entries, err := os.ReadDir(adminRoot)
	if err != nil {
		t.Fatalf("read admin root %q: %v", adminRoot, err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(adminRoot, name)
		parsed, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %q: %v", path, parseErr)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}
			methodName := strings.TrimSpace(fn.Name.Name)
			if strings.HasPrefix(methodName, "handle") {
				t.Fatalf("root admin package must not define area route method %q (%s); move route ownership into modules/<area>", methodName, path)
			}
		}
	}
}

func TestModuleAdapterShimIsRemoved(t *testing.T) {
	t.Parallel()

	adminRoot := filepath.Dir(moduleRoot(t))
	path := filepath.Join(adminRoot, rootAdapterFileName)
	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("legacy adapter shim %q exists; remove root module adapter wiring", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat %q: %v", path, err)
	}
}

func TestAreaModulesFollowTemplate(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	for _, name := range discoverAreaModules(t, root) {
		area := filepath.Join(root, name)
		for _, file := range requiredModuleFiles {
			path := filepath.Join(area, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("module %q missing required file %q: %v", name, file, err)
			}
		}
	}
}

func TestRootAdminRuntimeDoesNotContainLegacyHandlerFiles(t *testing.T) {
	t.Parallel()

	adminRoot := filepath.Dir(moduleRoot(t))
	for _, name := range []string{
		"handler_helpers.go",
		"handler_client_helpers.go",
		"handler_formatters.go",
		"handler_catalog_dispatch.go",
		"dashboard_activity_service.go",
		"campaign_character_helpers.go",
		"dashboard_event_helpers.go",
		"participants_invites_helpers.go",
		"events_helpers.go",
		"scenario_helpers.go",
	} {
		path := filepath.Join(adminRoot, name)
		_, err := os.Stat(path)
		if err == nil {
			t.Fatalf("root runtime file %q exists; move legacy handler/helper behavior into module packages", path)
		}
		if !os.IsNotExist(err) {
			t.Fatalf("stat %q: %v", path, err)
		}
	}
}

func TestRootAdminDoesNotContainModuleScopedTestFiles(t *testing.T) {
	t.Parallel()

	adminRoot := filepath.Dir(moduleRoot(t))
	for _, name := range []string{
		"handler_test.go",
		"helpers_test.go",
		"campaign_character_helpers_test.go",
		"dashboard_activity_service_test.go",
		"dashboard_activity_service_legacy_test.go",
		"dashboard_event_helpers_test.go",
		"events_helpers_test.go",
		"handler_catalog_dispatch_test.go",
		"handler_client_helpers_test.go",
		"handler_formatters_test.go",
		"handler_helpers_test.go",
		"participants_invites_helpers_test.go",
		"scenario_helpers_test.go",
	} {
		path := filepath.Join(adminRoot, name)
		_, err := os.Stat(path)
		if err == nil {
			t.Fatalf("root module-scoped test file %q exists; keep module tests under internal/services/admin/modules/<area>", path)
		}
		if !os.IsNotExist(err) {
			t.Fatalf("stat %q: %v", path, err)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	return filepath.Dir(file)
}

func discoverAreaModules(t *testing.T, root string) []string {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read module root %q: %v", root, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func moduleGoFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read module dir %q: %v", dir, err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	return files
}
