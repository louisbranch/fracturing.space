package modules

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectedSmallModulesKeepCompositionOwnedAppServices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modulePath  string
		configField string
		configType  string
		appPackage  string
		constructor string
		composePath string
		composeFunc string
	}{
		{
			name:        "discovery",
			modulePath:  "discovery/module.go",
			configField: "Service",
			configType:  "discoveryapp.Service",
			appPackage:  "discoveryapp",
			constructor: "NewService",
			composePath: "discovery/composition.go",
			composeFunc: "Compose",
		},
		{
			name:        "profile",
			modulePath:  "profile/module.go",
			configField: "Service",
			configType:  "profileapp.Service",
			appPackage:  "profileapp",
			constructor: "NewService",
			composePath: "profile/composition.go",
			composeFunc: "Compose",
		},
		{
			name:        "dashboard",
			modulePath:  "dashboard/module.go",
			configField: "Service",
			configType:  "dashboardapp.Service",
			appPackage:  "dashboardapp",
			constructor: "NewService",
			composePath: "dashboard/composition.go",
			composeFunc: "Compose",
		},
		{
			name:        "notifications",
			modulePath:  "notifications/module.go",
			configField: "Service",
			configType:  "notificationsapp.Service",
			appPackage:  "notificationsapp",
			constructor: "NewService",
			composePath: "notifications/composition.go",
			composeFunc: "Compose",
		},
		{
			name:        "invite",
			modulePath:  "invite/module.go",
			configField: "Service",
			configType:  "inviteapp.Service",
			appPackage:  "inviteapp",
			constructor: "NewService",
			composePath: "invite/composition.go",
			composeFunc: "Compose",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assertStructFieldType(t, tc.modulePath, "Config", tc.configField, tc.configType)
			assertStructLacksField(t, tc.modulePath, "Config", "Gateway")
			assertFuncDoesNotCallSelector(t, tc.modulePath, "Mount", tc.appPackage, tc.constructor)
			assertFuncCallsSelector(t, tc.composePath, tc.composeFunc, tc.appPackage, tc.constructor)
		})
	}
}

func TestSettingsCapabilitySplitStaysConstructorVisible(t *testing.T) {
	t.Parallel()

	assertStructFieldType(t, "settings/module.go", "Config", "Services", "handlerServices")
	assertStructFieldType(t, "settings/module.go", "Config", "Availability", "settingsSurfaceAvailability")
	assertFuncDoesNotCallSelector(t, "settings/module.go", "Mount", "settingsapp", "NewAccountService")
	assertFuncDoesNotCallSelector(t, "settings/module.go", "Mount", "settingsapp", "NewAIService")
	assertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsgateway", "NewAccountGateway")
	assertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsgateway", "NewAIGateway")
	assertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsapp", "NewAccountService")
	assertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsapp", "NewAIService")

	assertTypeExists(t, "settings/app/service.go", "AccountServiceConfig")
	assertTypeExists(t, "settings/app/service.go", "AIServiceConfig")
	assertTypeExists(t, "settings/app/service.go", "accountService")
	assertTypeExists(t, "settings/app/service.go", "aiService")
	assertTypeMissing(t, "settings/app/service.go", "ServiceConfig")
	assertFuncExists(t, "settings/app/service.go", "NewAccountService")
	assertFuncExists(t, "settings/app/service.go", "NewAIService")
	assertFuncMissing(t, "settings/app/service.go", "NewService")
}

func TestPublicAuthSurfaceServicesStaySplitEndToEnd(t *testing.T) {
	t.Parallel()

	assertStructFieldType(t, "publicauth/module.go", "Config", "PageService", "publicauthapp.PageService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "SessionService", "publicauthapp.SessionService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "PasskeyService", "publicauthapp.PasskeyService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "Recovery", "publicauthapp.RecoveryService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "Principal", "principal.PrincipalResolver")
	assertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewPageService")
	assertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewSessionService")
	assertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewPasskeyService")
	assertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewRecoveryService")

	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Pages", "publicauthapp.PageService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Session", "publicauthapp.SessionService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Passkeys", "publicauthapp.PasskeyService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Recovery", "publicauthapp.RecoveryService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Principal", "principal.PrincipalResolver")
	assertFuncCallsSelector(t, "publicauth/handlers.go", "newHandlers", "publichandler", "NewBaseFromPrincipal")

	assertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewPageService")
	assertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewSessionService")
	assertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewPasskeyService")
	assertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewRecoveryService")
	assertFuncExists(t, "publicauth/composition.go", "ComposeSurfaceSet")

	assertTypeExists(t, "publicauth/app/service.go", "pageService")
	assertTypeExists(t, "publicauth/app/service.go", "sessionService")
	assertTypeExists(t, "publicauth/app/service.go", "passkeyService")
	assertTypeExists(t, "publicauth/app/service.go", "recoveryService")
	assertFuncExists(t, "publicauth/app/service.go", "NewPageService")
	assertFuncExists(t, "publicauth/app/service.go", "NewSessionService")
	assertFuncExists(t, "publicauth/app/service.go", "NewPasskeyService")
	assertFuncExists(t, "publicauth/app/service.go", "NewRecoveryService")
	assertFuncMissing(t, "publicauth/app/service.go", "NewService")
	assertTypeMissing(t, "publicauth/app/types.go", "Service")
	assertTypeMissing(t, "publicauth/app/types.go", "Gateway")
}

func TestPublicAuthUsesSharedRedirectAndPrincipalSeams(t *testing.T) {
	t.Parallel()

	assertFileCallsSelector(t, "publicauth/handlers_pages.go", "redirectpath", "ResolveSafe")
	assertFileCallsSelector(t, "publicauth/app/service.go", "redirectpath", "ResolveSafe")
	assertFuncCallsMethod(t, "publicauth/handlers_session.go", "redirectAuthenticatedToApp", "h", "IsViewerSignedIn")
	assertStructFieldType(t, "publicauth/composition.go", "CompositionConfig", "Principal", "principal.PrincipalResolver")
	assertImportAbsent(t, "publicauth/handlers.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
}

func TestModuleDependenciesStayNestedByOwnedArea(t *testing.T) {
	t.Parallel()

	assertStructFieldType(t, "module.go", "Dependencies", "Campaigns", "CampaignDependencies")
	assertStructFieldType(t, "module.go", "Dependencies", "Profile", "ProfileDependencies")
	assertStructFieldType(t, "module.go", "Dependencies", "Settings", "SettingsDependencies")
	assertStructLacksField(t, "module.go", "Dependencies", "SocialClient")
	assertStructLacksField(t, "module.go", "Dependencies", "ProfileSocialClient")
	assertStructLacksField(t, "module.go", "Dependencies", "SettingsSocialClient")
}

func TestRegistryUsesAreaOwnedCompositionEntrypoints(t *testing.T) {
	t.Parallel()

	assertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "publicauth", "ComposeSurfaceSet")
	assertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "discovery", "ComposePublic")
	assertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "profile", "ComposePublic")
	assertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "invite", "ComposePublic")
	assertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "publicauthgateway", "NewGRPCGateway")
	assertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "profilegateway", "NewGRPCGateway")
	assertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "invitegateway", "NewGRPCGateway")
	assertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "dashboardsync", "New")

	assertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "modulehandler", "NewBaseFromPrincipal")
	assertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "dashboard", "ComposeProtected")
	assertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "settings", "ComposeProtected")
	assertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "campaigns", "ComposeProtected")
	assertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "notifications", "ComposeProtected")
	assertFuncDoesNotCallSelector(t, "registry_protected.go", "buildProtectedModules", "dashboardgateway", "NewGRPCGateway")
	assertFuncDoesNotCallSelector(t, "registry_protected.go", "buildProtectedModules", "settingsgateway", "NewGRPCGateway")
	assertFuncDoesNotCallSelector(t, "registry_protected.go", "buildProtectedModules", "campaigngateway", "NewGRPCGateway")

	assertFuncCallsIdent(t, "registry.go", "Build", "newSharedServices")
	assertFuncDoesNotCallSelector(t, "registry.go", "Build", "dashboardsync", "New")
}

func TestRequestResolutionUsesPrincipalSeam(t *testing.T) {
	t.Parallel()

	assertTypeExists(t, "../principal/requeststate.go", "PageResolver")
	assertTypeExists(t, "../principal/requeststate.go", "PrincipalResolver")
	assertTypeExists(t, "../principal/callbacks.go", "ViewerFunc")
	assertTypeExists(t, "../principal/callbacks.go", "SignedInFunc")
	assertTypeExists(t, "../principal/callbacks.go", "UserIDFunc")
	assertTypeExists(t, "../principal/callbacks.go", "LanguageFunc")
	assertFuncExists(t, "../principal/requeststate.go", "NewBaseFromPageResolver")
	assertFuncExists(t, "../principal/requeststate.go", "ResolveLocalizedPage")
	assertTypeMissing(t, "../module/module.go", "ResolveViewer")
	assertTypeMissing(t, "../module/module.go", "ResolveSignedIn")
	assertTypeMissing(t, "../module/module.go", "ResolveUserID")
	assertTypeMissing(t, "../module/module.go", "ResolveLanguage")
	assertImportPresent(t, "../platform/modulehandler/modulehandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	assertImportPresent(t, "../platform/publichandler/publichandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	assertImportPresent(t, "../platform/pagerender/pagerender.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	assertImportPresent(t, "../platform/weberror/weberror.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	assertImportAbsent(t, "../platform/modulehandler/modulehandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
	assertImportAbsent(t, "../platform/publichandler/publichandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
	assertImportAbsent(t, "../platform/pagerender/pagerender.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
	assertImportAbsent(t, "../platform/weberror/weberror.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
}

func parseFile(t *testing.T, path string) *ast.File {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return parsed
}

func exprString(t *testing.T, expr ast.Expr) string {
	t.Helper()

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), expr); err != nil {
		t.Fatalf("print expr: %v", err)
	}
	return buf.String()
}

func assertStructFieldType(t *testing.T, path, structName, fieldName, wantType string) {
	t.Helper()

	fields := structFieldTypes(t, path, structName)
	got, ok := fields[fieldName]
	if !ok {
		t.Fatalf("%s missing field %s.%s", path, structName, fieldName)
	}
	if got != wantType {
		t.Fatalf("%s field %s.%s = %q, want %q", path, structName, fieldName, got, wantType)
	}
}

func assertStructLacksField(t *testing.T, path, structName, fieldName string) {
	t.Helper()

	fields := structFieldTypes(t, path, structName)
	if got, ok := fields[fieldName]; ok {
		t.Fatalf("%s unexpectedly has field %s.%s of type %q", path, structName, fieldName, got)
	}
}

func structFieldTypes(t *testing.T, path, structName string) map[string]string {
	t.Helper()

	parsed := parseFile(t, path)
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil || typeSpec.Name.Name != structName {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				t.Fatalf("%s type %s is not a struct", path, structName)
			}
			fields := make(map[string]string)
			for _, field := range st.Fields.List {
				fieldType := exprString(t, field.Type)
				for _, name := range field.Names {
					fields[name.Name] = fieldType
				}
			}
			return fields
		}
	}

	t.Fatalf("%s missing struct type %s", path, structName)
	return nil
}

func assertTypeExists(t *testing.T, path, typeName string) {
	t.Helper()
	if !hasType(path, typeName) {
		t.Fatalf("%s missing type %s", path, typeName)
	}
}

func assertTypeMissing(t *testing.T, path, typeName string) {
	t.Helper()
	if hasType(path, typeName) {
		t.Fatalf("%s unexpectedly defines type %s", path, typeName)
	}
}

func hasType(path, typeName string) bool {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name != nil && typeSpec.Name.Name == typeName {
				return true
			}
		}
	}
	return false
}

func assertFuncExists(t *testing.T, path, funcName string) {
	t.Helper()
	if !hasFunc(path, funcName) {
		t.Fatalf("%s missing func %s", path, funcName)
	}
}

func assertFuncMissing(t *testing.T, path, funcName string) {
	t.Helper()
	if hasFunc(path, funcName) {
		t.Fatalf("%s unexpectedly defines func %s", path, funcName)
	}
}

func hasFunc(path, funcName string) bool {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name != nil && fn.Name.Name == funcName {
			return true
		}
	}
	return false
}

func assertFuncCallsSelector(t *testing.T, path, funcName, recvName, selector string) {
	t.Helper()

	if !funcCalls(path, funcName, func(call *ast.CallExpr) bool {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != selector {
			return false
		}
		ident, ok := sel.X.(*ast.Ident)
		return ok && ident.Name == recvName
	}) {
		t.Fatalf("%s %s does not call %s.%s", path, funcName, recvName, selector)
	}
}

func assertFuncDoesNotCallSelector(t *testing.T, path, funcName, recvName, selector string) {
	t.Helper()

	if funcCalls(path, funcName, func(call *ast.CallExpr) bool {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != selector {
			return false
		}
		ident, ok := sel.X.(*ast.Ident)
		return ok && ident.Name == recvName
	}) {
		t.Fatalf("%s %s still calls %s.%s", path, funcName, recvName, selector)
	}
}

func assertFuncCallsMethod(t *testing.T, path, funcName, recvName, selector string) {
	t.Helper()

	assertFuncCallsSelector(t, path, funcName, recvName, selector)
}

func assertFileCallsSelector(t *testing.T, path, recvName, selector string) {
	t.Helper()

	parsed := parseFile(t, path)
	found := false
	ast.Inspect(parsed, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != selector {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != recvName {
			return true
		}
		found = true
		return false
	})
	if !found {
		t.Fatalf("%s does not call %s.%s", path, recvName, selector)
	}
}

func assertFuncCallsIdent(t *testing.T, path, funcName, identName string) {
	t.Helper()

	if !funcCalls(path, funcName, func(call *ast.CallExpr) bool {
		ident, ok := call.Fun.(*ast.Ident)
		return ok && ident.Name == identName
	}) {
		t.Fatalf("%s %s does not call %s", path, funcName, identName)
	}
}

func funcCalls(path, funcName string, match func(*ast.CallExpr) bool) bool {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != funcName || fn.Body == nil {
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if match(call) {
				found = true
				return false
			}
			return true
		})
		return found
	}
	return false
}

func assertImportPresent(t *testing.T, path, importPath string) {
	t.Helper()
	if !hasImport(t, path, importPath) {
		t.Fatalf("%s missing import %s", path, importPath)
	}
}

func assertImportAbsent(t *testing.T, path, importPath string) {
	t.Helper()
	if hasImport(t, path, importPath) {
		t.Fatalf("%s unexpectedly imports %s", path, importPath)
	}
}

func hasImport(t *testing.T, path, importPath string) bool {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports for %s: %v", path, err)
	}
	for _, imp := range parsed.Imports {
		if strings.Trim(imp.Path.Value, "\"") == importPath {
			return true
		}
	}
	return false
}

func TestWebModulePlaybookMatchesCurrentPublicModuleContract(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "guides", "web-module-playbook.md"))
	if err != nil {
		t.Fatalf("ReadFile(web-module-playbook.md) error = %v", err)
	}
	body := string(content)
	for _, required := range []string{"publichandler.Base", "principal.PrincipalResolver"} {
		if !strings.Contains(body, required) {
			t.Fatalf("web-module-playbook.md missing %q", required)
		}
	}
}
