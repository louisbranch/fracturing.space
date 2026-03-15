// Package main validates shared i18n catalogs for consistency and safety.
package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	i18ncatalog "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
)

const (
	notificationPayloadImportPath = "github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
	platformI18NImportPath        = "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	flashImportPath               = "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	netHTTPImportPath             = "net/http"
	allowRawCommentPrefix         = "i18n:allow-raw"
)

type codedError struct {
	code int
	err  error
}

func (e codedError) Error() string {
	return e.err.Error()
}

func (e codedError) Unwrap() error {
	return e.err
}

func withExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return codedError{code: code, err: err}
}

func exitCode(err error) int {
	var codeErr codedError
	if errors.As(err, &codeErr) {
		return codeErr.code
	}
	return 1
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(exitCode(err))
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	var baseLocale string
	var strictMissing bool
	var repoRoot string
	flags := flag.NewFlagSet("i18ncheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&baseLocale, "base-locale", i18ncatalog.BaseLocale, "base locale used as translation source of truth")
	flags.BoolVar(&strictMissing, "strict-missing", false, "fail when non-base locales are missing base keys")
	flags.StringVar(&repoRoot, "repo-root", ".", "repository root used for static user-copy seam checks")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return withExitCode(err, 2)
	}

	bundle, err := i18ncatalog.LoadEmbedded()
	if err != nil {
		fmt.Fprintf(stderr, "load i18n catalogs: %v\n", err)
		return withExitCode(err, 1)
	}

	if !bundle.HasLocale(baseLocale) {
		err := fmt.Errorf("base locale %q is missing from catalogs", baseLocale)
		fmt.Fprintf(stderr, "%v\n", err)
		return withExitCode(err, 1)
	}

	failures := make([]string, 0, 32)
	warnings := make([]string, 0, 32)

	for _, tag := range platformi18n.SupportedTags() {
		locale := tag.String()
		if !bundle.HasLocale(locale) {
			failures = append(failures, fmt.Sprintf("supported locale %q is missing from catalogs", locale))
		}
	}

	baseMessages := bundle.LocaleMessages(baseLocale)
	baseKeys := sortedKeys(baseMessages)
	locales := bundle.Locales()
	for _, locale := range locales {
		if locale == baseLocale {
			continue
		}
		localeMessages := bundle.LocaleMessages(locale)
		missing := 0
		extra := 0
		for _, key := range baseKeys {
			baseValue := baseMessages[key]
			translatedValue, ok := localeMessages[key]
			if !ok {
				missing++
				if strictMissing {
					failures = append(failures, fmt.Sprintf("locale %s missing key %s", locale, key))
				}
				continue
			}
			if !equalTokenMultiset(printfTokens(baseValue), printfTokens(translatedValue)) {
				failures = append(failures, fmt.Sprintf("locale %s key %s has mismatched printf placeholders", locale, key))
			}
			if !equalTokenMultiset(templateTokens(baseValue), templateTokens(translatedValue)) {
				failures = append(failures, fmt.Sprintf("locale %s key %s has mismatched template placeholders", locale, key))
			}
		}
		for key := range localeMessages {
			if _, ok := baseMessages[key]; !ok {
				extra++
			}
		}
		warnings = append(warnings, fmt.Sprintf("locale %s: missing=%d extra=%d", locale, missing, extra))
	}

	sourceFailures, err := collectSourceFailures(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "scan source seams: %v\n", err)
		return withExitCode(err, 1)
	}
	failures = append(failures, sourceFailures...)

	for _, line := range warnings {
		fmt.Fprintln(stdout, line)
	}
	if len(failures) > 0 {
		for _, line := range failures {
			fmt.Fprintf(stderr, "i18n check failure: %s\n", line)
		}
		return withExitCode(errors.New("i18n check failure"), 1)
	}
	fmt.Fprintln(stdout, "i18n catalog check passed")
	return nil
}

func printfTokens(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	verbs := map[byte]struct{}{
		'b': {}, 'c': {}, 'd': {}, 'e': {}, 'E': {}, 'f': {}, 'F': {}, 'g': {}, 'G': {},
		'o': {}, 'O': {}, 'p': {}, 'q': {}, 's': {}, 't': {}, 'T': {}, 'U': {}, 'v': {},
		'x': {}, 'X': {},
	}
	out := make([]string, 0, 4)
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		if i+1 < len(value) && value[i+1] == '%' {
			i++
			continue
		}
		j := i + 1
		for j < len(value) {
			if _, ok := verbs[value[j]]; ok {
				out = append(out, value[i:j+1])
				i = j
				break
			}
			if value[j] == '%' {
				break
			}
			j++
		}
	}
	sort.Strings(out)
	return out
}

func templateTokens(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	out := make([]string, 0, 4)
	for {
		start := strings.Index(value, "{{")
		if start < 0 {
			break
		}
		value = value[start+2:]
		end := strings.Index(value, "}}")
		if end < 0 {
			break
		}
		token := strings.TrimSpace(value[:end])
		value = value[end+2:]
		if strings.HasPrefix(token, ".") {
			name := strings.TrimSpace(strings.TrimPrefix(token, "."))
			if name != "" {
				out = append(out, name)
			}
		}
	}
	sort.Strings(out)
	return out
}

func equalTokenMultiset(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sortedKeys(entries map[string]string) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type seamType string

const (
	seamUnknown       seamType = ""
	seamInAppPayload  seamType = "notificationpayload.InAppPayload"
	seamPayloadFact   seamType = "notificationpayload.PayloadFact"
	seamPayloadAction seamType = "notificationpayload.PayloadAction"
	seamFlashNotice   seamType = "flash.Notice"
)

type fileAnalyzer struct {
	fset            *token.FileSet
	relPath         string
	file            *ast.File
	imports         map[string]string
	suppressedLines map[int]struct{}
	issues          []string
}

func collectSourceFailures(repoRoot string) ([]string, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}
	fset := token.NewFileSet()
	issues := make([]string, 0, 16)
	if err := filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		fileAnalyzer := newFileAnalyzer(fset, absRoot, path, file)
		issues = append(issues, fileAnalyzer.analyze()...)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(issues)
	return issues, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".tmp", "node_modules", "vendor":
		return true
	default:
		return strings.HasPrefix(name, ".") && name != "."
	}
}

func newFileAnalyzer(fset *token.FileSet, root, path string, file *ast.File) *fileAnalyzer {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		relPath = path
	}
	return &fileAnalyzer{
		fset:            fset,
		relPath:         filepath.ToSlash(relPath),
		file:            file,
		imports:         fileImportMap(file),
		suppressedLines: collectSuppressedLines(fset, file),
		issues:          make([]string, 0, 4),
	}
}

func (a *fileAnalyzer) analyze() []string {
	ast.Inspect(a.file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CompositeLit:
			a.analyzeCompositeLit(typed)
		case *ast.FuncDecl:
			a.analyzeFuncBody(typed.Body)
			return false
		case *ast.FuncLit:
			a.analyzeFuncBody(typed.Body)
			return false
		case *ast.CallExpr:
			a.analyzeCallExpr(typed)
		}
		return true
	})
	return a.issues
}

func (a *fileAnalyzer) analyzeFuncBody(body *ast.BlockStmt) {
	if body == nil {
		return
	}
	tracked := make(map[string]seamType)
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CompositeLit:
			a.analyzeCompositeLit(typed)
		case *ast.AssignStmt:
			a.trackAssignStmt(tracked, typed)
			a.analyzeAssignStmt(tracked, typed)
		case *ast.CallExpr:
			a.analyzeCallExpr(typed)
		case *ast.DeclStmt:
			genDecl, ok := typed.Decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				return true
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				a.trackValueSpec(tracked, valueSpec)
			}
		case *ast.RangeStmt:
			if typed.Value != nil {
				if ident, ok := typed.Value.(*ast.Ident); ok && typed.Tok == token.DEFINE {
					delete(tracked, ident.Name)
				}
			}
		}
		return true
	})
}

func (a *fileAnalyzer) analyzeCompositeLit(lit *ast.CompositeLit) {
	switch seamTypeFromTypeExpr(lit.Type, a.imports) {
	case seamInAppPayload:
		a.analyzeCompositeFields(lit, map[string]func(ast.Expr){
			"Title": func(expr ast.Expr) {
				a.reportRawNotificationField(expr, "raw notification payload Title")
			},
			"Body": func(expr ast.Expr) {
				a.reportRawNotificationField(expr, "raw notification payload Body")
			},
			"Facts": func(expr ast.Expr) {
				a.reportRawNestedNotificationLabels(expr)
			},
			"Actions": func(expr ast.Expr) {
				a.reportRawNestedNotificationLabels(expr)
			},
		})
	case seamPayloadFact:
		a.analyzeCompositeFields(lit, map[string]func(ast.Expr){
			"Label": func(expr ast.Expr) {
				a.reportRawNotificationField(expr, "raw notification payload Label")
			},
		})
	case seamPayloadAction:
		a.analyzeCompositeFields(lit, map[string]func(ast.Expr){
			"Label": func(expr ast.Expr) {
				a.reportRawNotificationField(expr, "raw notification payload Label")
			},
		})
	case seamFlashNotice:
		a.analyzeCompositeFields(lit, map[string]func(ast.Expr){
			"Message": func(expr ast.Expr) {
				a.reportIssue(expr, "raw flash notice Message")
			},
			"Key": func(expr ast.Expr) {
				if looksLikeRawKeyExpr(expr, a.imports) {
					a.reportIssue(expr, "raw flash notice Key")
				}
			},
		})
	}
}

func (a *fileAnalyzer) analyzeCompositeFields(lit *ast.CompositeLit, handlers map[string]func(ast.Expr)) {
	for _, elt := range lit.Elts {
		keyValue, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := keyValue.Key.(*ast.Ident)
		if !ok {
			continue
		}
		handler, ok := handlers[key.Name]
		if !ok {
			continue
		}
		handler(keyValue.Value)
	}
}

func (a *fileAnalyzer) trackValueSpec(tracked map[string]seamType, spec *ast.ValueSpec) {
	explicitType := seamTypeFromTypeExpr(spec.Type, a.imports)
	for idx, name := range spec.Names {
		if name == nil {
			continue
		}
		if explicitType != seamUnknown {
			tracked[name.Name] = explicitType
			continue
		}
		if idx < len(spec.Values) {
			if inferred := seamTypeFromExpr(spec.Values[idx], a.imports); inferred != seamUnknown {
				tracked[name.Name] = inferred
			}
		}
	}
}

func (a *fileAnalyzer) trackAssignStmt(tracked map[string]seamType, stmt *ast.AssignStmt) {
	if len(stmt.Lhs) != len(stmt.Rhs) {
		return
	}
	for idx, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		inferred := seamTypeFromExpr(stmt.Rhs[idx], a.imports)
		if inferred == seamUnknown {
			if stmt.Tok == token.DEFINE {
				delete(tracked, ident.Name)
			}
			continue
		}
		tracked[ident.Name] = inferred
	}
}

func (a *fileAnalyzer) analyzeAssignStmt(tracked map[string]seamType, stmt *ast.AssignStmt) {
	if len(stmt.Lhs) != len(stmt.Rhs) {
		return
	}
	for idx, lhs := range stmt.Lhs {
		selector, ok := lhs.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			continue
		}
		seam := tracked[ident.Name]
		if seam == seamUnknown {
			continue
		}
		rhs := stmt.Rhs[idx]
		switch seam {
		case seamInAppPayload:
			if selector.Sel.Name == "Title" || selector.Sel.Name == "Body" {
				a.reportRawNotificationField(rhs, "raw notification payload "+selector.Sel.Name)
			}
		case seamPayloadFact, seamPayloadAction:
			if selector.Sel.Name == "Label" {
				a.reportRawNotificationField(rhs, "raw notification payload Label")
			}
		case seamFlashNotice:
			switch selector.Sel.Name {
			case "Message":
				a.reportIssue(rhs, "raw flash notice Message")
			case "Key":
				if looksLikeRawKeyExpr(rhs, a.imports) {
					a.reportIssue(rhs, "raw flash notice Key")
				}
			}
		}
	}
}

func (a *fileAnalyzer) analyzeCallExpr(call *ast.CallExpr) {
	if isImportedSelector(call.Fun, a.imports, flashImportPath, "NoticeSuccess") {
		if len(call.Args) > 0 && looksLikeRawKeyExpr(call.Args[0], a.imports) {
			a.reportIssue(call.Args[0], "raw flash notice key passed to NoticeSuccess")
		}
	}
	if !strings.Contains(a.relPath, "internal/services/web/") {
		return
	}
	if !isImportedSelector(call.Fun, a.imports, netHTTPImportPath, "Error") {
		return
	}
	if len(call.Args) < 2 {
		return
	}
	if containsErrorMethodCall(call.Args[1]) {
		a.reportIssue(call.Args[1], "request-facing err.Error() fallback passed to http.Error")
	}
}

func (a *fileAnalyzer) reportRawNotificationField(expr ast.Expr, message string) {
	if looksLikeRawNotificationExpr(expr, a.imports) {
		a.reportIssue(expr, message)
	}
}

func (a *fileAnalyzer) reportRawNestedNotificationLabels(expr ast.Expr) {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return
	}
	for _, elt := range lit.Elts {
		child, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}
		a.analyzeCompositeFields(child, map[string]func(ast.Expr){
			"Label": func(value ast.Expr) {
				a.reportRawNotificationField(value, "raw notification payload Label")
			},
		})
	}
}

func (a *fileAnalyzer) reportIssue(node ast.Node, message string) {
	if node == nil || a.isSuppressed(node) {
		return
	}
	line := a.fset.Position(node.Pos()).Line
	a.issues = append(a.issues, fmt.Sprintf("%s:%d %s", a.relPath, line, message))
}

func (a *fileAnalyzer) isSuppressed(node ast.Node) bool {
	if node == nil {
		return false
	}
	line := a.fset.Position(node.Pos()).Line
	if _, ok := a.suppressedLines[line]; ok {
		return true
	}
	_, ok := a.suppressedLines[line-1]
	return ok
}

func collectSuppressedLines(fset *token.FileSet, file *ast.File) map[int]struct{} {
	lines := make(map[int]struct{})
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if !strings.HasPrefix(text, allowRawCommentPrefix) {
				continue
			}
			reason := strings.TrimSpace(strings.TrimPrefix(text, allowRawCommentPrefix))
			if reason == "" {
				continue
			}
			lines[fset.Position(comment.Pos()).Line] = struct{}{}
		}
	}
	return lines
}

func fileImportMap(file *ast.File) map[string]string {
	imports := make(map[string]string, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = path
	}
	return imports
}

func seamTypeFromExpr(expr ast.Expr, imports map[string]string) seamType {
	switch typed := expr.(type) {
	case *ast.CompositeLit:
		return seamTypeFromTypeExpr(typed.Type, imports)
	case *ast.UnaryExpr:
		if typed.Op == token.AND {
			return seamTypeFromExpr(typed.X, imports)
		}
	case *ast.ParenExpr:
		return seamTypeFromExpr(typed.X, imports)
	}
	return seamUnknown
}

func seamTypeFromTypeExpr(expr ast.Expr, imports map[string]string) seamType {
	switch typed := expr.(type) {
	case *ast.StarExpr:
		return seamTypeFromTypeExpr(typed.X, imports)
	case *ast.ParenExpr:
		return seamTypeFromTypeExpr(typed.X, imports)
	case *ast.SelectorExpr:
		pkgIdent, ok := typed.X.(*ast.Ident)
		if !ok {
			return seamUnknown
		}
		switch imports[pkgIdent.Name] {
		case notificationPayloadImportPath:
			switch typed.Sel.Name {
			case "InAppPayload":
				return seamInAppPayload
			case "PayloadFact":
				return seamPayloadFact
			case "PayloadAction":
				return seamPayloadAction
			}
		case flashImportPath:
			if typed.Sel.Name == "Notice" {
				return seamFlashNotice
			}
		}
	}
	return seamUnknown
}

func looksLikeRawNotificationExpr(expr ast.Expr, imports map[string]string) bool {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return false
		}
		value, ok := stringLiteralValue(typed)
		return ok && !looksLikeLocalizationKeyLiteral(value)
	case *ast.BinaryExpr:
		return typed.Op == token.ADD
	case *ast.CallExpr:
		if isNewCopyRefCall(typed.Fun, imports) {
			if len(typed.Args) == 0 {
				return true
			}
			if literal, ok := firstStringLiteral(typed.Args[0]); ok {
				return !looksLikeLocalizationKeyLiteral(literal)
			}
			return false
		}
		return isStringFormattingCall(typed.Fun, imports)
	case *ast.CompositeLit:
		if !isImportedSelector(typed.Type, imports, platformI18NImportPath, "CopyRef") {
			return false
		}
		return copyRefCompositeHasRawKey(typed)
	case *ast.ParenExpr:
		return looksLikeRawNotificationExpr(typed.X, imports)
	}
	return false
}

func looksLikeRawKeyExpr(expr ast.Expr, imports map[string]string) bool {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return false
		}
		value, ok := stringLiteralValue(typed)
		return ok && !looksLikeLocalizationKeyLiteral(value)
	case *ast.BinaryExpr:
		return typed.Op == token.ADD
	case *ast.CallExpr:
		if isNewCopyRefCall(typed.Fun, imports) {
			if len(typed.Args) == 0 {
				return true
			}
			if literal, ok := firstStringLiteral(typed.Args[0]); ok {
				return !looksLikeLocalizationKeyLiteral(literal)
			}
			return false
		}
		return isStringFormattingCall(typed.Fun, imports)
	case *ast.ParenExpr:
		return looksLikeRawKeyExpr(typed.X, imports)
	}
	return false
}

func copyRefCompositeHasRawKey(lit *ast.CompositeLit) bool {
	for _, elt := range lit.Elts {
		keyValue, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := keyValue.Key.(*ast.Ident)
		if !ok || key.Name != "Key" {
			continue
		}
		if literal, ok := firstStringLiteral(keyValue.Value); ok {
			return !looksLikeLocalizationKeyLiteral(literal)
		}
	}
	return false
}

func looksLikeLocalizationKeyLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	if strings.ContainsAny(value, "!?\"'") {
		return false
	}
	if value == strings.ToUpper(value) {
		hasSeparator := false
		for _, r := range value {
			if r == '_' || r == '-' || r == '.' {
				hasSeparator = true
				continue
			}
			if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				return false
			}
		}
		return hasSeparator
	}
	hasSeparator := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
			hasSeparator = true
		default:
			return false
		}
	}
	return hasSeparator
}

func containsErrorMethodCall(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Error" || len(call.Args) != 0 {
			return true
		}
		found = true
		return false
	})
	return found
}

func isStringFormattingCall(fun ast.Expr, imports map[string]string) bool {
	selector, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	switch imports[pkgIdent.Name] {
	case "fmt":
		switch selector.Sel.Name {
		case "Sprintf", "Sprint", "Sprintln":
			return true
		}
	case "strings":
		return selector.Sel.Name == "Join"
	}
	return false
}

func isNewCopyRefCall(fun ast.Expr, imports map[string]string) bool {
	return isImportedSelector(fun, imports, platformI18NImportPath, "NewCopyRef")
}

func isImportedSelector(expr ast.Expr, imports map[string]string, pkgPath, name string) bool {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		ident, ok := typed.X.(*ast.Ident)
		if !ok {
			return false
		}
		return imports[ident.Name] == pkgPath && typed.Sel.Name == name
	case *ast.ParenExpr:
		return isImportedSelector(typed.X, imports, pkgPath, name)
	}
	return false
}

func firstStringLiteral(expr ast.Expr) (string, bool) {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		return stringLiteralValue(typed)
	case *ast.ParenExpr:
		return firstStringLiteral(typed.X)
	}
	return "", false
}

func stringLiteralValue(lit *ast.BasicLit) (string, bool) {
	if lit == nil || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}
