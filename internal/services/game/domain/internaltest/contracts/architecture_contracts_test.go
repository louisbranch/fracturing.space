package contracts

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestGamePackagesHaveDocGo(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	gameRoot := filepath.Join(repoRoot, "internal", "services", "game")

	dirs, err := dirsWithNonTestGoFiles(gameRoot)
	if err != nil {
		t.Fatalf("list game package dirs: %v", err)
	}

	var missing []string
	for _, dir := range dirs {
		docPath := filepath.Join(dir, "doc.go")
		if _, err := os.Stat(docPath); err != nil {
			if os.IsNotExist(err) {
				rel, relErr := filepath.Rel(repoRoot, dir)
				if relErr != nil {
					t.Fatalf("relative dir %s: %v", dir, relErr)
				}
				missing = append(missing, filepath.ToSlash(rel))
				continue
			}
			t.Fatalf("stat %s: %v", docPath, err)
		}
	}

	if len(missing) == 0 {
		return
	}
	sort.Strings(missing)
	t.Fatalf("game package docs missing doc.go:\n%s", strings.Join(missing, "\n"))
}

func TestGameDocGoUsesPackageCommentConvention(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	gameRoot := filepath.Join(repoRoot, "internal", "services", "game")

	dirs, err := dirsWithNonTestGoFiles(gameRoot)
	if err != nil {
		t.Fatalf("list game package dirs: %v", err)
	}

	fset := token.NewFileSet()
	var violations []string
	for _, dir := range dirs {
		docPath := filepath.Join(dir, "doc.go")
		file, err := parser.ParseFile(fset, docPath, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", docPath, err)
		}

		if file.Doc == nil {
			rel := relPathOrFatal(t, repoRoot, docPath)
			violations = append(violations, fmt.Sprintf("%s: missing package comment", rel))
			continue
		}
		docText := strings.TrimSpace(file.Doc.Text())
		prefix := "Package " + file.Name.Name
		if docText != prefix && !strings.HasPrefix(docText, prefix+" ") {
			rel := relPathOrFatal(t, repoRoot, docPath)
			violations = append(violations, fmt.Sprintf("%s: package comment must begin with %q", rel, prefix))
		}
	}

	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("game package comment convention violations:\n%s", strings.Join(violations, "\n"))
}

func TestDomainImportsRespectArchitectureContracts(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	modulePath, err := modulePathFromGoMod(repoRoot)
	if err != nil {
		t.Fatalf("read module path: %v", err)
	}
	domainRoot := filepath.Join(repoRoot, "internal", "services", "game", "domain")
	dirs, err := dirsWithNonTestGoFiles(domainRoot)
	if err != nil {
		t.Fatalf("list domain package dirs: %v", err)
	}

	allowStorageImports := map[string]struct{}{
		modulePath + "/internal/services/game/domain/bridge/daggerheart":                     {},
		modulePath + "/internal/services/game/domain/bridge/daggerheart/internal/projection": {},
		modulePath + "/internal/services/game/domain/bridge/manifest":                        {},
	}
	allowProtoImports := map[string]struct{}{}

	apiPrefix := modulePath + "/internal/services/game/api/"
	appPrefix := modulePath + "/internal/services/game/app"
	storageSQLitePrefix := modulePath + "/internal/services/game/storage/sqlite"
	storageContractsPath := modulePath + "/internal/services/game/storage"
	protoPrefix := modulePath + "/api/gen/go/"

	var violations []string
	for _, dir := range dirs {
		relDir, err := filepath.Rel(repoRoot, dir)
		if err != nil {
			t.Fatalf("relative dir %s: %v", dir, err)
		}
		importPath := modulePath + "/" + filepath.ToSlash(relDir)
		imports, err := importsFromDir(dir)
		if err != nil {
			t.Fatalf("imports from %s: %v", dir, err)
		}
		for _, imp := range imports {
			switch {
			case strings.HasPrefix(imp, apiPrefix):
				violations = append(violations, fmt.Sprintf("%s imports transport package %s", filepath.ToSlash(relDir), imp))
			case imp == appPrefix || strings.HasPrefix(imp, appPrefix+"/"):
				violations = append(violations, fmt.Sprintf("%s imports app runtime package %s", filepath.ToSlash(relDir), imp))
			case strings.HasPrefix(imp, storageSQLitePrefix):
				violations = append(violations, fmt.Sprintf("%s imports concrete sqlite package %s", filepath.ToSlash(relDir), imp))
			case imp == storageContractsPath:
				if _, ok := allowStorageImports[importPath]; !ok {
					violations = append(violations, fmt.Sprintf("%s imports storage contracts package %s without allowlist", filepath.ToSlash(relDir), imp))
				}
			case strings.HasPrefix(imp, protoPrefix):
				if _, ok := allowProtoImports[importPath]; !ok {
					violations = append(violations, fmt.Sprintf("%s imports proto package %s without allowlist", filepath.ToSlash(relDir), imp))
				}
			}
		}
	}

	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("domain import architecture violations:\n%s", strings.Join(violations, "\n"))
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", ".."))
}

func modulePathFromGoMod(repoRoot string) (string, error) {
	goModPath := filepath.Join(repoRoot, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", goModPath, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
			if modulePath == "" {
				return "", fmt.Errorf("module path is empty in %s", goModPath)
			}
			return modulePath, nil
		}
	}
	return "", fmt.Errorf("module declaration not found in %s", goModPath)
}

func dirsWithNonTestGoFiles(root string) ([]string, error) {
	dirs := make(map[string]struct{})
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		dirs[filepath.Dir(path)] = struct{}{}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	out := make([]string, 0, len(dirs))
	for dir := range dirs {
		out = append(out, dir)
	}
	sort.Strings(out)
	return out, nil
}

func importsFromDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	importSet := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil, fmt.Errorf("parse imports %s: %w", path, err)
		}
		for _, spec := range file.Imports {
			trimmed := strings.Trim(spec.Path.Value, "\"")
			importSet[trimmed] = struct{}{}
		}
	}
	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports, nil
}

func relPathOrFatal(t *testing.T, repoRoot, path string) string {
	t.Helper()
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("relative path %s: %v", path, err)
	}
	return filepath.ToSlash(rel)
}
