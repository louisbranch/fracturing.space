//go:build integration
// +build integration

package integration

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestProjectionStoreWritesAreEventDriven(t *testing.T) {
	config := &packages.Config{
		Mode:  packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps,
		Tests: false,
		Dir:   integrationRepoRoot(t),
	}
	storagePkgs, err := packages.Load(config, "./internal/services/game/storage")
	if err != nil {
		t.Fatalf("load storage package: %v", err)
	}
	if packages.PrintErrors(storagePkgs) > 0 {
		t.Fatalf("storage package load errors")
	}
	if len(storagePkgs) == 0 {
		t.Fatal("storage package not found")
	}
	storagePkg := storagePkgs[0]

	targetPkgs, err := packages.Load(config, projectionWriteGuardrailPatterns()...)
	if err != nil {
		t.Fatalf("load target packages: %v", err)
	}
	if packages.PrintErrors(targetPkgs) > 0 {
		t.Fatalf("target package load errors")
	}

	storeInterfaces := []*types.Interface{
		lookupInterface(t, storagePkg, "CampaignStore"),
		lookupInterface(t, storagePkg, "ParticipantStore"),
		lookupInterface(t, storagePkg, "ClaimIndexStore"),
		lookupInterface(t, storagePkg, "CharacterStore"),
		lookupInterface(t, storagePkg, "SessionStore"),
		lookupInterface(t, storagePkg, "InviteStore"),
		lookupInterface(t, storagePkg, "DaggerheartStore"),
		lookupInterface(t, storagePkg, "SnapshotStore"),
		lookupInterface(t, storagePkg, "CampaignForkStore"),
	}

	forbiddenMethods := map[string]struct{}{
		"Put":                            {},
		"PutParticipant":                 {},
		"DeleteParticipant":              {},
		"PutCharacter":                   {},
		"DeleteCharacter":                {},
		"PutSession":                     {},
		"EndSession":                     {},
		"PutInvite":                      {},
		"UpdateInviteStatus":             {},
		"PutParticipantClaim":            {},
		"DeleteParticipantClaim":         {},
		"PutDaggerheartCharacterProfile": {},
		"PutDaggerheartCharacterState":   {},
		"PutDaggerheartSnapshot":         {},
		"PutDaggerheartCountdown":        {},
		"DeleteDaggerheartCountdown":     {},
		"PutDaggerheartAdversary":        {},
		"DeleteDaggerheartAdversary":     {},
		"PutSnapshot":                    {},
		"SetCampaignForkMetadata":        {},
	}

	var violations []string
	for _, pkg := range targetPkgs {
		if isProjectionWriteGuardrailIgnoredPackage(pkg.PkgPath) {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if _, ok := forbiddenMethods[sel.Sel.Name]; !ok {
					return true
				}
				receiverType := pkg.TypesInfo.TypeOf(sel.X)
				if receiverType == nil {
					return true
				}
				if !implementsAnyStore(receiverType, storeInterfaces) {
					return true
				}
				position := pkg.Fset.Position(sel.Pos())
				violations = append(violations, formatProjectionWriteViolation(pkg.PkgPath, file, sel, position.String()))
				return true
			})
		}
	}

	if len(violations) > 0 {
		formatted := make([]string, 0, len(violations))
		for _, violation := range violations {
			formatted = append(formatted, "- "+filepath.ToSlash(violation))
		}
		t.Fatalf("direct projection store writes must go through event appliers:\n%s", strings.Join(formatted, "\n"))
	}
}

func formatProjectionWriteViolation(pkgPath string, file *ast.File, sel *ast.SelectorExpr, position string) string {
	if sel == nil || sel.Sel == nil {
		return fmt.Sprintf("%s: direct projection store write", position)
	}
	location := strings.TrimSpace(position)
	if location == "" {
		location = "<unknown>"
	}
	pkgPath = filepath.ToSlash(strings.TrimSpace(pkgPath))
	if pkgPath == "" {
		pkgPath = "<unknown-package>"
	}
	funcName := enclosingFunctionName(file, sel.Pos())
	if strings.TrimSpace(funcName) == "" {
		funcName = "<unknown-function>"
	}
	return fmt.Sprintf("%s: %s %s calls %s", location, pkgPath, funcName, sel.Sel.Name)
}

func enclosingFunctionName(file *ast.File, pos token.Pos) string {
	if file == nil {
		return ""
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if pos < fn.Pos() || pos > fn.End() {
			continue
		}
		if fn.Recv == nil || len(fn.Recv.List) == 0 {
			return fn.Name.Name
		}
		recvName := receiverTypeName(fn.Recv.List[0].Type)
		if recvName == "" {
			return fn.Name.Name
		}
		return recvName + "." + fn.Name.Name
	}
	return ""
}

func receiverTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return receiverTypeName(typed.X)
	case *ast.IndexExpr:
		return receiverTypeName(typed.X)
	case *ast.IndexListExpr:
		return receiverTypeName(typed.X)
	case *ast.SelectorExpr:
		if typed.Sel != nil {
			return typed.Sel.Name
		}
		return ""
	default:
		return ""
	}
}

func lookupInterface(t *testing.T, pkg *packages.Package, name string) *types.Interface {
	obj := pkg.Types.Scope().Lookup(name)
	if obj == nil {
		t.Fatalf("storage interface %s not found", name)
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		t.Fatalf("storage type %s is not an interface", name)
	}
	return iface
}

func integrationRepoRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("go.mod not found")
		}
		wd = parent
	}
}

func implementsAnyStore(typ types.Type, interfaces []*types.Interface) bool {
	if typ == nil {
		return false
	}
	for _, iface := range interfaces {
		if types.Implements(typ, iface) {
			return true
		}
		if types.Implements(types.NewPointer(typ), iface) {
			return true
		}
	}
	return false
}

func TestProjectionWriteGuardrailScopes(t *testing.T) {
	patterns := projectionWriteGuardrailPatterns()
	if len(patterns) == 0 {
		t.Fatal("expected at least one package pattern")
	}
	found := false
	for _, pattern := range patterns {
		if pattern == "./internal/services/game/..." {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected scan scope to include ./internal/services/game/..., got %v", patterns)
	}
}

func TestProjectionWriteGuardrailIgnoresAuthorizedPackages(t *testing.T) {
	if !isProjectionWriteGuardrailIgnoredPackage("github.com/louisbranch/fracturing.space/internal/services/game/projection") {
		t.Fatal("expected projection package to be ignored")
	}
	if !isProjectionWriteGuardrailIgnoredPackage("github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite") {
		t.Fatal("expected storage package to be ignored")
	}
	if isProjectionWriteGuardrailIgnoredPackage("github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game") {
		t.Fatal("expected API package to be scanned")
	}
}

func projectionWriteGuardrailPatterns() []string {
	return []string{
		"./internal/services/game/...",
	}
}

func isProjectionWriteGuardrailIgnoredPackage(pkgPath string) bool {
	path := filepath.ToSlash(strings.TrimSpace(pkgPath))
	if path == "" {
		return false
	}
	return strings.Contains(path, "/internal/services/game/projection") ||
		strings.Contains(path, "/internal/services/game/storage")
}
