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
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.GenDecl:
					if typed.Tok == token.CONST {
						defs.Events = append(defs.Events, parseConstDecl(typed, fset, root, owner)...)
					}
					if typed.Tok == token.TYPE {
						for _, spec := range typed.Specs {
							typeSpec, ok := spec.(*ast.TypeSpec)
							if !ok {
								continue
							}
							structType, ok := typeSpec.Type.(*ast.StructType)
							if !ok {
								continue
							}
							name := typeSpec.Name.Name
							if !strings.HasSuffix(name, "Payload") {
								continue
							}
							payload := payloadDef{
								Owner:     owner,
								Name:      name,
								DefinedAt: formatPosition(fset.Position(typeSpec.Pos()), root),
								Fields:    parsePayloadFields(structType.Fields, fset),
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

func parseConstDecl(decl *ast.GenDecl, fset *token.FileSet, root, owner string) []eventDef {
	var events []eventDef
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if valueSpec.Type == nil {
			continue
		}
		typeString := exprString(fset, valueSpec.Type)
		if typeString != "Type" && typeString != "event.Type" {
			continue
		}
		for idx, name := range valueSpec.Names {
			valueExpr := selectValueExpr(valueSpec.Values, idx)
			if valueExpr == nil {
				continue
			}
			lit, ok := valueExpr.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
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
