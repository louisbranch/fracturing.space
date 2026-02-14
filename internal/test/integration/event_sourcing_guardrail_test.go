//go:build integration
// +build integration

package integration

import (
	"go/ast"
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

	apiPkgs, err := packages.Load(config, "./internal/services/game/api/grpc/...")
	if err != nil {
		t.Fatalf("load api packages: %v", err)
	}
	if packages.PrintErrors(apiPkgs) > 0 {
		t.Fatalf("api package load errors")
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
	for _, pkg := range apiPkgs {
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
				violations = append(violations, position.String()+": direct projection store write")
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
