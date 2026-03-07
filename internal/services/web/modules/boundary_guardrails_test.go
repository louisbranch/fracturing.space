package modules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

func TestProtectedModuleCompatibilityShimsAreRemoved(t *testing.T) {
	t.Parallel()

	paths := []string{
		"settings/service.go",
		"settings/gateway_grpc.go",
		"settings/gateway_unavailable.go",
		"notifications/service.go",
		"notifications/gateway_grpc.go",
		"notifications/gateway_unavailable.go",
		"dashboard/service.go",
		"dashboard/gateway_grpc.go",
		"dashboard/gateway_unavailable.go",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			_, err := os.Stat(path)
			if err == nil {
				t.Fatalf("%q exists; protected-module compatibility shim should be removed after cutover", path)
			}
			if !os.IsNotExist(err) {
				t.Fatalf("Stat(%q) error = %v", path, err)
			}
		})
	}
}

func TestProtectedModuleRootsWireAppServicesDirectly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path   string
		pkg    string
		method string
	}{
		{path: "settings/module.go", pkg: "settingsapp", method: "NewService"},
		{path: "notifications/module.go", pkg: "notificationsapp", method: "NewService"},
		{path: "dashboard/module.go", pkg: "dashboardapp", method: "NewService"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			assertMountCallsSelector(t, tc.path, tc.pkg, tc.method)
		})
	}
}

func TestProfileCompatibilityShimsAreRemoved(t *testing.T) {
	t.Parallel()

	paths := []string{
		"profile/service.go",
		"profile/gateway_grpc.go",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			_, err := os.Stat(path)
			if err == nil {
				t.Fatalf("%q exists; profile compatibility shim should be removed after cutover", path)
			}
			if !os.IsNotExist(err) {
				t.Fatalf("Stat(%q) error = %v", path, err)
			}
		})
	}
}

func TestProfileRootWiresAppServiceDirectly(t *testing.T) {
	t.Parallel()
	assertMountCallsSelector(t, "profile/module.go", "profileapp", "NewService")
}

func TestPublicAuthCompatibilityShimsAreRemoved(t *testing.T) {
	t.Parallel()

	paths := []string{
		"publicauth/service.go",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			_, err := os.Stat(path)
			if err == nil {
				t.Fatalf("%q exists; publicauth compatibility shim should be removed after cutover", path)
			}
			if !os.IsNotExist(err) {
				t.Fatalf("Stat(%q) error = %v", path, err)
			}
		})
	}
}

func TestPublicAuthRootWiresAppServiceDirectly(t *testing.T) {
	t.Parallel()
	assertMountCallsSelector(t, "publicauth/module.go", "publicauthapp", "NewService")
}

func TestModuleDependenciesSocialContractsAreSplit(t *testing.T) {
	t.Parallel()

	fields := dependenciesStructFields(t, "module.go")
	if _, exists := fields["SocialClient"]; exists {
		t.Fatalf("Dependencies still exposes legacy SocialClient; expected nested Profile.SocialClient + Settings.SocialClient")
	}
	for _, required := range []string{"Profile", "Settings"} {
		if _, exists := fields[required]; !exists {
			t.Fatalf("Dependencies missing required field %q", required)
		}
	}
	for _, forbidden := range []string{"ProfileSocialClient", "SettingsSocialClient"} {
		if _, exists := fields[forbidden]; exists {
			t.Fatalf("Dependencies still exposes deprecated flat field %q", forbidden)
		}
	}
}

func TestRegistryWiresSplitSocialContracts(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"registry.go", "registry_public.go", "registry_protected.go"} {
		if hasDepsSelector(t, path, "SocialClient") {
			t.Fatalf("%s uses deps.SocialClient; expected split social contract fields", path)
		}
	}
	assertRegistryGatewayCallUsesNestedDepField(t, "registry_public.go", "profilegateway", "NewGRPCGateway", 0, "Profile", "SocialClient")
	assertRegistryGatewayCallUsesNestedDepField(t, "registry_protected.go", "settingsgateway", "NewGRPCGateway", 0, "Settings", "SocialClient")
}

func assertMountCallsSelector(t *testing.T, path, pkgName, methodName string) {
	t.Helper()

	parsed := parseFile(t, path)
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "Mount" || fn.Body == nil {
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != methodName {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != pkgName {
				return true
			}
			found = true
			return false
		})
		if !found {
			t.Fatalf("%s Mount does not call %s.%s", path, pkgName, methodName)
		}
		return
	}

	t.Fatalf("%s missing Mount function", path)
}

func dependenciesStructFields(t *testing.T, path string) map[string]struct{} {
	t.Helper()

	parsed := parseFile(t, path)
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil || typeSpec.Name.Name != "Dependencies" {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				t.Fatalf("Dependencies type in %s is not a struct", path)
			}
			fields := make(map[string]struct{})
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					fields[name.Name] = struct{}{}
				}
			}
			return fields
		}
	}

	t.Fatalf("%s missing Dependencies struct", path)
	return nil
}

func hasDepsSelector(t *testing.T, path, selector string) bool {
	t.Helper()

	parsed := parseFile(t, path)
	found := false
	ast.Inspect(parsed, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != selector {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "deps" {
			return true
		}
		found = true
		return false
	})
	return found
}

func assertRegistryGatewayCallUsesDepField(t *testing.T, path, pkgName, methodName string, argIndex int, depField string) {
	t.Helper()

	parsed := parseFile(t, path)
	found := false
	ast.Inspect(parsed, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != methodName {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != pkgName {
			return true
		}
		if argIndex >= len(call.Args) {
			return true
		}
		argSel, ok := call.Args[argIndex].(*ast.SelectorExpr)
		if !ok || argSel.Sel == nil || argSel.Sel.Name != depField {
			return true
		}
		argIdent, ok := argSel.X.(*ast.Ident)
		if !ok || argIdent.Name != "deps" {
			return true
		}
		found = true
		return false
	})
	if !found {
		t.Fatalf("%s does not call %s.%s with deps.%s in argument %d", path, pkgName, methodName, depField, argIndex)
	}
}

func assertRegistryGatewayCallUsesNestedDepField(t *testing.T, path, pkgName, methodName string, argIndex int, depFields ...string) {
	t.Helper()

	parsed := parseFile(t, path)
	found := false
	ast.Inspect(parsed, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != methodName {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != pkgName {
			return true
		}
		if argIndex >= len(call.Args) {
			return true
		}
		if matchesDepsSelectorChain(call.Args[argIndex], depFields...) {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Fatalf("%s does not call %s.%s with deps.%v in argument %d", path, pkgName, methodName, depFields, argIndex)
	}
}

func matchesDepsSelectorChain(expr ast.Expr, depFields ...string) bool {
	if len(depFields) == 0 {
		return false
	}
	current := expr
	for i := len(depFields) - 1; i >= 0; i-- {
		sel, ok := current.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != depFields[i] {
			return false
		}
		current = sel.X
	}
	ident, ok := current.(*ast.Ident)
	return ok && ident.Name == "deps"
}

func parseFile(t *testing.T, path string) *ast.File {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return parsed
}
