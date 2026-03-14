package modules

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

const (
	moduleImportPrefix     = "github.com/louisbranch/fracturing.space/internal/services/admin/modules/"
	legacyAreaImportPrefix = "github.com/louisbranch/fracturing.space/internal/services/admin/module/"
)

var requiredModuleFiles = []string{
	"module.go",
	"handlers.go",
	"routes.go",
	"module_test.go",
}

// sharedModulePackages are shared utility packages under modules/ that any
// module may import. These are not treated as sibling modules.
var sharedModulePackages = []string{
	"eventview",
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
				if isSharedModulePackage(path) {
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

func TestModulesDoNotContainNilClientChecks(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	for _, name := range discoverAreaModules(t, root) {
		dir := filepath.Join(root, name)
		for _, file := range moduleGoFiles(t, dir) {
			f, err := os.Open(file)
			if err != nil {
				t.Fatalf("open %s: %v", file, err)
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
			lineNo := 0
			for scanner.Scan() {
				lineNo++
				line := scanner.Text()
				trimmed := strings.TrimSpace(line)
				if strings.Contains(trimmed, "Client == nil") || strings.Contains(trimmed, "Client != nil") {
					t.Fatalf("%s:%d contains nil-client check %q; clients are guaranteed non-nil via unavailable stubs", file, lineNo, trimmed)
				}
			}
			if err := scanner.Err(); err != nil {
				t.Fatalf("scan %s: %v", file, err)
			}
		}
	}
}

func TestEnsureClientsFillsAllFields(t *testing.T) {
	t.Parallel()

	input := BuildInput{}
	input.ensureClients()

	v := reflect.ValueOf(input)
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := typ.Field(i)
		fv := v.Field(i)
		if !strings.HasSuffix(field.Name, "Client") {
			continue
		}
		if fv.Kind() == reflect.Interface && fv.IsNil() {
			t.Fatalf("ensureClients() did not fill BuildInput.%s", field.Name)
		}
	}
}

func isSharedModulePackage(importPath string) bool {
	for _, pkg := range sharedModulePackages {
		if strings.HasPrefix(importPath, moduleImportPrefix+pkg) {
			return true
		}
	}
	return false
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
		if isSharedModulePackage(moduleImportPrefix + name) {
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
