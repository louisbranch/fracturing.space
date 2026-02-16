package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type eventDef struct {
	Owner     string
	Name      string
	Value     string
	DefinedAt string
}

type payloadField struct {
	Name    string
	Type    string
	JSONTag string
}

type payloadDef struct {
	Owner     string
	Name      string
	DefinedAt string
	Fields    []payloadField
}

type packageDefs struct {
	Events   []eventDef
	Payloads map[string]payloadDef
}

func main() {
	var outPath string
	var rootFlag string
	flag.StringVar(&outPath, "out", "docs/events/event-catalog.md", "output path for the catalog")
	flag.StringVar(&rootFlag, "root", "", "repo root (defaults to locating go.mod)")
	flag.Parse()

	root, err := resolveRoot(rootFlag)
	if err != nil {
		fatal(err)
	}
	output := outPath
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, outPath)
	}

	coreDir := filepath.Join(root, "internal/services/game/domain/campaign/event")
	daggerheartDir := filepath.Join(root, "internal/services/game/domain/systems/daggerheart")

	coreDefs, err := parsePackage(coreDir, root, "Core")
	if err != nil {
		fatal(err)
	}
	daggerheartDefs, err := parsePackage(daggerheartDir, root, "Daggerheart")
	if err != nil {
		fatal(err)
	}

	emitters, err := scanEmitters(filepath.Join(root, "internal/services/game"), root)
	if err != nil {
		fatal(err)
	}

	content, err := renderCatalog([]packageDefs{coreDefs, daggerheartDefs}, emitters)
	if err != nil {
		fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fatal(fmt.Errorf("create output dir: %w", err))
	}
	if err := os.WriteFile(output, []byte(content), 0o644); err != nil {
		fatal(fmt.Errorf("write catalog: %w", err))
	}
}

func resolveRoot(flagRoot string) (string, error) {
	if flagRoot != "" {
		return filepath.Clean(flagRoot), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working dir: %w", err)
	}
	return findModuleRoot(wd)
}

func findModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found above %s", start)
}

func parsePackage(dir, root, owner string) (packageDefs, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.AllErrors)
	if err != nil {
		return packageDefs{}, fmt.Errorf("parse %s: %w", dir, err)
	}
	defs := packageDefs{Payloads: make(map[string]payloadDef)}
	for _, pkg := range pkgs {
		files := make([]*ast.File, 0, len(pkg.Files))
		typeSpecs := make(map[string]ast.Expr)
		for _, file := range pkg.Files {
			files = append(files, file)
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					typeSpecs[typeSpec.Name.Name] = typeSpec.Type
				}
			}
		}
		for _, file := range pkg.Files {
			importAliases := parseImportAliases(file)
			ast.Inspect(file, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.GenDecl:
					if typed.Tok == token.CONST {
						defs.Events = append(defs.Events, parseConstDecl(typed, fset, root, owner, importAliases)...)
					}
					if typed.Tok == token.TYPE {
						for _, spec := range typed.Specs {
							typeSpec, ok := spec.(*ast.TypeSpec)
							if !ok {
								continue
							}
							name := typeSpec.Name.Name
							if !strings.HasSuffix(name, "Payload") {
								continue
							}
							payloadFields, definedAt, ok := parsePayloadFromTypeExpr(typeSpec.Type, fset, importAliases, root, typeSpecs, map[string]struct{}{name: {}})
							if !ok {
								continue
							}
							payload := payloadDef{
								Owner:     owner,
								Name:      name,
								DefinedAt: definedAt,
								Fields:    payloadFields,
							}
							defs.Payloads[name] = payload
						}
					}
				}
				return true
			})
		}
	}
	for i := range defs.Events {
		defs.Events[i].Owner = owner
	}
	return defs, nil
}

func parseConstDecl(decl *ast.GenDecl, fset *token.FileSet, root, owner string, importAliases map[string]string) []eventDef {
	var events []eventDef
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		typeString := ""
		if valueSpec.Type != nil {
			typeString = exprString(fset, valueSpec.Type)
			if typeString != "Type" && typeString != "event.Type" {
				continue
			}
		}
		for idx, name := range valueSpec.Names {
			if !strings.HasPrefix(name.Name, "Type") && !strings.HasPrefix(name.Name, "EventType") {
				continue
			}
			if !isExported(name.Name) {
				continue
			}
			valueExpr := selectValueExpr(valueSpec.Values, idx)
			if valueExpr == nil {
				continue
			}
			if valueSpec.Type == nil {
				if _, ok := valueExpr.(*ast.SelectorExpr); !ok {
					continue
				}
			}
			value, ok := constantValue(valueExpr, importAliases, root)
			if !ok {
				continue
			}
			events = append(events, eventDef{
				Owner:     owner,
				Name:      name.Name,
				Value:     value,
				DefinedAt: formatPosition(fset.Position(name.Pos()), root),
			})
		}
	}
	return events
}

func parsePayloadFields(fields *ast.FieldList, fset *token.FileSet) []payloadField {
	if fields == nil {
		return nil
	}
	results := make([]payloadField, 0)
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			continue
		}
		typeString := exprString(fset, field.Type)
		jsonTag := ""
		if field.Tag != nil {
			tagValue, err := strconv.Unquote(field.Tag.Value)
			if err == nil {
				tag := reflect.StructTag(tagValue)
				if jsonValue := tag.Get("json"); jsonValue != "" {
					jsonTag = fmt.Sprintf("json:\"%s\"", jsonValue)
				}
			}
		}
		for _, name := range field.Names {
			results = append(results, payloadField{
				Name:    name.Name,
				Type:    typeString,
				JSONTag: jsonTag,
			})
		}
	}
	return results
}

func parsePayloadFromTypeExpr(expr ast.Expr, fset *token.FileSet, importAliases map[string]string, root string, typeSpecs map[string]ast.Expr, seen map[string]struct{}) ([]payloadField, string, bool) {
	switch typed := expr.(type) {
	case *ast.StructType:
		return parsePayloadFields(typed.Fields, fset), formatPosition(fset.Position(typed.Pos()), root), true
	case *ast.SelectorExpr:
		return parseImportedPayload(typed, importAliases, root)
	case *ast.Ident:
		name := typed.Name
		if _, ok := seen[name]; ok {
			return nil, "", false
		}
		target, ok := typeSpecs[name]
		if !ok {
			return nil, "", false
		}
		next := make(map[string]struct{}, len(seen)+1)
		for k := range seen {
			next[k] = struct{}{}
		}
		next[name] = struct{}{}
		return parsePayloadFromTypeExpr(target, fset, importAliases, root, typeSpecs, next)
	default:
		return nil, "", false
	}
}

func parseImportedPayload(sel *ast.SelectorExpr, importAliases map[string]string, root string) ([]payloadField, string, bool) {
	alias := sel.X
	ident, ok := alias.(*ast.Ident)
	if !ok {
		return nil, "", false
	}
	importPath, ok := importAliases[ident.Name]
	if !ok {
		return nil, "", false
	}
	importDir := resolveImportDir(importPath, root)
	if importDir == "" {
		return nil, "", false
	}
	return parsePayloadFromPackage(importDir, sel.Sel.Name, root)
}

func parsePayloadFromPackage(dir, typeName, root string) ([]payloadField, string, bool) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.AllErrors)
	if err != nil {
		return nil, "", false
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || typeSpec.Name.Name != typeName {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					return parsePayloadFields(structType.Fields, fset), formatPosition(fset.Position(typeSpec.Pos()), root), true
				}
			}
		}
	}
	return nil, "", false
}

func parseImportAliases(file *ast.File) map[string]string {
	aliases := make(map[string]string)
	for _, imp := range file.Imports {
		pathValue, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(pathValue)
		if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "." && imp.Name.Name != "_" {
			name = imp.Name.Name
		}
		aliases[name] = pathValue
	}
	// Default import name is package identifier.
	return aliases
}

func resolveImportDir(importPath, root string) string {
	const modulePath = "github.com/louisbranch/fracturing.space/"
	if !strings.HasPrefix(importPath, modulePath) {
		return ""
	}
	relPath := strings.TrimPrefix(importPath, modulePath)
	candidate := filepath.Join(root, filepath.FromSlash(relPath))
	if _, err := os.Stat(candidate); err != nil {
		return ""
	}
	return candidate
}

func constantValue(expr ast.Expr, importAliases map[string]string, root string) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		if sel, ok := expr.(*ast.SelectorExpr); ok {
			return constFromSelector(sel, importAliases, root)
		}
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func constFromSelector(expr *ast.SelectorExpr, importAliases map[string]string, root string) (string, bool) {
	alias := expr.X
	ident, ok := alias.(*ast.Ident)
	if !ok {
		return "", false
	}
	importPath, ok := importAliases[ident.Name]
	if !ok {
		return "", false
	}
	importDir := resolveImportDir(importPath, root)
	if importDir == "" {
		return "", false
	}
	return constFromPackage(importDir, expr.Sel.Name)
}

func constFromPackage(dir, constantName string) (string, bool) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.AllErrors)
	if err != nil {
		return "", false
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.CONST {
					continue
				}
				for _, spec := range gen.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range valueSpec.Names {
						if name.Name != constantName {
							continue
						}
						valueExpr := selectValueExpr(valueSpec.Values, i)
						if valueExpr == nil {
							continue
						}
						value, ok := constantValue(valueExpr, nil, dir)
						if ok {
							return value, true
						}
					}
				}
			}
		}
	}
	return "", false
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

func selectValueExpr(values []ast.Expr, index int) ast.Expr {
	if len(values) == 0 {
		return nil
	}
	if len(values) == 1 {
		return values[0]
	}
	if index < len(values) {
		return values[index]
	}
	return nil
}

func scanEmitters(dir, root string) (map[string][]string, error) {
	emitters := make(map[string][]string)
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		relPath, _ := filepath.Rel(root, path)
		ast.Inspect(file, func(node ast.Node) bool {
			lit, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			selector, ok := lit.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok || ident.Name != "event" || selector.Sel.Name != "Event" {
				return true
			}
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "Type" {
					continue
				}
				eventName := eventNameFromExpr(kv.Value)
				if eventName == "" {
					continue
				}
				pos := fset.Position(kv.Value.Pos())
				location := fmt.Sprintf("%s:%d", relPath, pos.Line)
				emitters[eventName] = append(emitters[eventName], location)
			}
			return true
		})
		return nil
	}); err != nil {
		return nil, err
	}
	for key := range emitters {
		sort.Strings(emitters[key])
	}
	return emitters, nil
}

func eventNameFromExpr(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		return typed.Sel.Name
	case *ast.Ident:
		return typed.Name
	default:
		return ""
	}
}

func renderCatalog(packages []packageDefs, emitters map[string][]string) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.WriteString("title: \"Event Catalog\"\n")
	buf.WriteString("parent: \"Events\"\n")
	buf.WriteString("nav_order: 1\n")
	buf.WriteString("---\n\n")
	buf.WriteString("# Event Catalog\n\n")
	buf.WriteString("Generated by `go generate ./internal/services/game/domain/campaign/event`.\n\n")

	for _, pkg := range packages {
		if len(pkg.Events) == 0 {
			continue
		}
		sort.Slice(pkg.Events, func(i, j int) bool {
			if pkg.Events[i].Value == pkg.Events[j].Value {
				return pkg.Events[i].Name < pkg.Events[j].Name
			}
			return pkg.Events[i].Value < pkg.Events[j].Value
		})
		buf.WriteString(fmt.Sprintf("## %s Events\n\n", pkg.Events[0].Owner))

		usedPayloads := make(map[string]struct{})
		for _, evt := range pkg.Events {
			payloadName := payloadNameForEvent(evt.Name, evt.Owner)
			payload, hasPayload := pkg.Payloads[payloadName]
			if hasPayload {
				usedPayloads[payloadName] = struct{}{}
			}

			buf.WriteString(fmt.Sprintf("### `%s` (`%s`)\n", evt.Value, evt.Name))
			buf.WriteString(fmt.Sprintf("- Defined at: `%s`\n", evt.DefinedAt))
			if hasPayload {
				buf.WriteString(fmt.Sprintf("- Payload: `%s` (`%s`)\n", payload.Name, payload.DefinedAt))
				if len(payload.Fields) > 0 {
					buf.WriteString("- Fields:\n")
					for _, field := range payload.Fields {
						label := field.Name
						if field.JSONTag != "" {
							label = fmt.Sprintf("%s (%s)", label, field.JSONTag)
						}
						buf.WriteString(fmt.Sprintf("  - `%s`: `%s`\n", label, field.Type))
					}
				}
			} else {
				buf.WriteString("- Payload: not found\n")
			}
			if locations, ok := emitters[evt.Name]; ok && len(locations) > 0 {
				buf.WriteString("- Emitters:\n")
				for _, location := range locations {
					buf.WriteString(fmt.Sprintf("  - `%s`\n", location))
				}
			}
			buf.WriteString("\n")
		}

		unmapped := unmappedPayloads(pkg.Payloads, usedPayloads)
		if len(unmapped) > 0 {
			buf.WriteString("### Unmapped Payloads\n")
			for _, payload := range unmapped {
				buf.WriteString(fmt.Sprintf("- `%s` (`%s`)\n", payload.Name, payload.DefinedAt))
			}
			buf.WriteString("\n")
		}
	}

	return buf.String(), nil
}

func payloadNameForEvent(eventName, owner string) string {
	if owner == "Core" && strings.HasPrefix(eventName, "Type") {
		return strings.TrimPrefix(eventName, "Type") + "Payload"
	}
	if owner == "Daggerheart" && strings.HasPrefix(eventName, "EventType") {
		return strings.TrimPrefix(eventName, "EventType") + "Payload"
	}
	return ""
}

func unmappedPayloads(payloads map[string]payloadDef, used map[string]struct{}) []payloadDef {
	if len(payloads) == 0 {
		return nil
	}
	result := make([]payloadDef, 0)
	for name, payload := range payloads {
		if _, ok := used[name]; ok {
			continue
		}
		result = append(result, payload)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, expr)
	return buf.String()
}

func formatPosition(pos token.Position, root string) string {
	rel, err := filepath.Rel(root, pos.Filename)
	if err != nil {
		rel = pos.Filename
	}
	return fmt.Sprintf("%s:%d", filepath.ToSlash(rel), pos.Line)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
