package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	modeDeclarations = "declarations"
	modePackages     = "packages"
	modeQuality      = "quality"
)

type declarationEntry struct {
	Path string
	Line int
	Kind string
	Name string
}

type packageEntry struct {
	ImportPath string
	Dir        string
}

type qualityEntry struct {
	Path   string
	Line   int
	Kind   string
	Name   string
	Phrase string
}

func main() {
	var mode string
	var baselinePath string
	var writeBaseline bool

	flag.StringVar(&mode, "mode", modeDeclarations, "check mode: declarations, packages, or quality")
	flag.StringVar(&baselinePath, "baseline", "", "optional baseline file used for staged rollout")
	flag.BoolVar(&writeBaseline, "write-baseline", false, "write current missing entries to stdout")
	flag.Parse()

	switch strings.ToLower(strings.TrimSpace(mode)) {
	case modeDeclarations:
		entries, err := missingDeclarationComments()
		if err != nil {
			fatalf("scan declaration comments: %v", err)
		}
		handleDeclarationResults(entries, baselinePath, writeBaseline)
	case modePackages:
		entries, err := missingPackageComments()
		if err != nil {
			fatalf("scan package comments: %v", err)
		}
		handlePackageResults(entries, baselinePath, writeBaseline)
	case modeQuality:
		entries, err := lowSignalCommentEntries()
		if err != nil {
			fatalf("scan comment quality: %v", err)
		}
		handleQualityResults(entries, baselinePath, writeBaseline)
	default:
		fatalf("unsupported mode %q (expected %q, %q, or %q)", mode, modeDeclarations, modePackages, modeQuality)
	}
}

func handleDeclarationResults(entries []declarationEntry, baselinePath string, writeBaseline bool) {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s:%d %s %s", entry.Path, entry.Line, entry.Kind, entry.Name))
	}
	sort.Strings(lines)

	if writeBaseline {
		for _, line := range lines {
			fmt.Println(line)
		}
		return
	}

	if strings.TrimSpace(baselinePath) == "" {
		if len(lines) == 0 {
			fmt.Println("webdoccheck: declaration comment check passed")
			return
		}
		fmt.Println("webdoccheck: missing declaration comments")
		for _, line := range lines {
			fmt.Println(line)
		}
		os.Exit(1)
	}

	baseline, err := readBaseline(baselinePath)
	if err != nil {
		fatalf("read baseline %q: %v", baselinePath, err)
	}

	currentSet := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		currentSet[line] = struct{}{}
	}

	newEntries := make([]string, 0)
	for _, line := range lines {
		if _, exists := baseline[line]; !exists {
			newEntries = append(newEntries, line)
		}
	}
	sort.Strings(newEntries)

	resolvedEntries := make([]string, 0)
	for line := range baseline {
		if _, exists := currentSet[line]; !exists {
			resolvedEntries = append(resolvedEntries, line)
		}
	}
	sort.Strings(resolvedEntries)

	if len(newEntries) > 0 {
		fmt.Printf("webdoccheck: %d new declaration comment violations (baseline %s)\n", len(newEntries), baselinePath)
		for _, line := range newEntries {
			fmt.Println(line)
		}
		os.Exit(1)
	}

	if len(resolvedEntries) > 0 {
		fmt.Printf("webdoccheck: declaration baseline can be ratcheted; %d entries resolved\n", len(resolvedEntries))
		fmt.Println("run: make web-doc-baseline-update")
	}
	fmt.Println("webdoccheck: declaration comment check passed")
}

func handlePackageResults(entries []packageEntry, baselinePath string, writeBaseline bool) {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s (%s)", entry.ImportPath, entry.Dir))
	}
	sort.Strings(lines)

	if writeBaseline {
		for _, line := range lines {
			fmt.Println(line)
		}
		return
	}

	if strings.TrimSpace(baselinePath) != "" {
		fatalf("-baseline is not supported in packages mode")
	}

	if len(lines) == 0 {
		fmt.Println("webdoccheck: package comment check passed")
		return
	}

	fmt.Println("webdoccheck: missing package comments")
	for _, line := range lines {
		fmt.Println(line)
	}
	os.Exit(1)
}

func handleQualityResults(entries []qualityEntry, baselinePath string, writeBaseline bool) {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s:%d %s %s (%s)", entry.Path, entry.Line, entry.Kind, entry.Name, entry.Phrase))
	}
	sort.Strings(lines)

	if writeBaseline {
		for _, line := range lines {
			fmt.Println(line)
		}
		return
	}

	if strings.TrimSpace(baselinePath) != "" {
		fatalf("-baseline is not supported in quality mode")
	}

	if len(lines) == 0 {
		fmt.Println("webdoccheck: comment quality check passed")
		return
	}

	fmt.Println("webdoccheck: low-signal declaration comments found")
	for _, line := range lines {
		fmt.Println(line)
	}
	os.Exit(1)
}

func missingDeclarationComments() ([]declarationEntry, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles,
		Dir:  ".",
		Fset: token.NewFileSet(),
	}
	pkgs, err := packages.Load(cfg, "./internal/services/web/...", "./internal/cmd/web")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("failed to load packages")
	}

	result := make([]declarationEntry, 0)
	for _, pkg := range pkgs {
		for _, goFile := range pkg.GoFiles {
			if skipSourceFile(goFile) {
				continue
			}
			parsed, err := parser.ParseFile(cfg.Fset, goFile, nil, parser.ParseComments)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", goFile, err)
			}
			rel := relPath(goFile)
			for _, decl := range parsed.Decls {
				switch typed := decl.(type) {
				case *ast.FuncDecl:
					if typed.Name == nil {
						continue
					}
					if hasCommentGroup(typed.Doc) {
						continue
					}
					kind := "func"
					if typed.Recv != nil {
						kind = "method"
					}
					result = append(result, declarationEntry{
						Path: rel,
						Line: cfg.Fset.Position(typed.Pos()).Line,
						Kind: kind,
						Name: typed.Name.Name,
					})
				case *ast.GenDecl:
					if typed.Tok != token.TYPE {
						continue
					}
					for _, spec := range typed.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok || typeSpec.Name == nil {
							continue
						}
						if hasCommentGroup(typeSpec.Doc) || hasCommentGroup(typed.Doc) {
							continue
						}
						result = append(result, declarationEntry{
							Path: rel,
							Line: cfg.Fset.Position(typeSpec.Pos()).Line,
							Kind: "type",
							Name: typeSpec.Name.Name,
						})
					}
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Path == result[j].Path {
			if result[i].Line == result[j].Line {
				if result[i].Kind == result[j].Kind {
					return result[i].Name < result[j].Name
				}
				return result[i].Kind < result[j].Kind
			}
			return result[i].Line < result[j].Line
		}
		return result[i].Path < result[j].Path
	})
	return result, nil
}

func missingPackageComments() ([]packageEntry, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports,
		Dir:  ".",
		Fset: token.NewFileSet(),
	}
	pkgs, err := packages.Load(cfg, "./internal/services/web/...", "./internal/cmd/web")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("failed to load packages")
	}

	result := make([]packageEntry, 0)
	for _, pkg := range pkgs {
		hasRelevantFiles := false
		hasPackageComment := false
		firstDir := ""
		for _, goFile := range pkg.GoFiles {
			if skipSourceFile(goFile) {
				continue
			}
			hasRelevantFiles = true
			if firstDir == "" {
				firstDir = relPath(filepath.Dir(goFile))
			}
			parsed, err := parser.ParseFile(cfg.Fset, goFile, nil, parser.ParseComments|parser.PackageClauseOnly)
			if err != nil {
				return nil, fmt.Errorf("parse package clause %s: %w", goFile, err)
			}
			if hasCommentGroup(parsed.Doc) {
				hasPackageComment = true
				break
			}
		}
		if !hasRelevantFiles || hasPackageComment {
			continue
		}
		result = append(result, packageEntry{
			ImportPath: pkg.PkgPath,
			Dir:        firstDir,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].ImportPath == result[j].ImportPath {
			return result[i].Dir < result[j].Dir
		}
		return result[i].ImportPath < result[j].ImportPath
	})
	return result, nil
}

func lowSignalCommentEntries() ([]qualityEntry, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles,
		Dir:  ".",
		Fset: token.NewFileSet(),
	}
	pkgs, err := packages.Load(cfg, "./internal/services/web/...", "./internal/cmd/web")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("failed to load packages")
	}

	result := make([]qualityEntry, 0)
	for _, pkg := range pkgs {
		for _, goFile := range pkg.GoFiles {
			if skipSourceFile(goFile) {
				continue
			}
			parsed, err := parser.ParseFile(cfg.Fset, goFile, nil, parser.ParseComments)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", goFile, err)
			}
			rel := relPath(goFile)
			for _, decl := range parsed.Decls {
				switch typed := decl.(type) {
				case *ast.FuncDecl:
					if typed.Name == nil || !hasCommentGroup(typed.Doc) {
						continue
					}
					if phrase, ok := lowSignalPhrase(typed.Doc.Text()); ok {
						kind := "func"
						if typed.Recv != nil {
							kind = "method"
						}
						result = append(result, qualityEntry{
							Path:   rel,
							Line:   cfg.Fset.Position(typed.Doc.Pos()).Line,
							Kind:   kind,
							Name:   typed.Name.Name,
							Phrase: phrase,
						})
					}
				case *ast.GenDecl:
					if typed.Tok != token.TYPE {
						continue
					}
					for _, spec := range typed.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok || typeSpec.Name == nil {
							continue
						}
						group := typeSpec.Doc
						if !hasCommentGroup(group) {
							group = typed.Doc
						}
						if !hasCommentGroup(group) {
							continue
						}
						if phrase, ok := lowSignalPhrase(group.Text()); ok {
							result = append(result, qualityEntry{
								Path:   rel,
								Line:   cfg.Fset.Position(group.Pos()).Line,
								Kind:   "type",
								Name:   typeSpec.Name.Name,
								Phrase: phrase,
							})
						}
					}
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Path == result[j].Path {
			if result[i].Line == result[j].Line {
				if result[i].Kind == result[j].Kind {
					return result[i].Name < result[j].Name
				}
				return result[i].Kind < result[j].Kind
			}
			return result[i].Line < result[j].Line
		}
		return result[i].Path < result[j].Path
	})
	return result, nil
}

func lowSignalPhrase(comment string) (string, bool) {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(comment)), " "))
	phrases := []string{
		"keeps this web package behavior centralized at one seam",
		"handles this route flow while keeping module transport logic localized",
	}
	for _, phrase := range phrases {
		if strings.Contains(normalized, phrase) {
			return phrase, true
		}
	}
	return "", false
}

func skipSourceFile(path string) bool {
	base := filepath.Base(path)
	if strings.HasSuffix(base, "_test.go") {
		return true
	}
	if strings.HasSuffix(base, "_templ.go") {
		return true
	}
	return false
}

func hasCommentGroup(group *ast.CommentGroup) bool {
	return group != nil && len(group.List) > 0
}

func relPath(path string) string {
	if path == "" {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	wd, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(wd, abs)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func readBaseline(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entries[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "webdoccheck: "+format+"\n", args...)
	os.Exit(2)
}
