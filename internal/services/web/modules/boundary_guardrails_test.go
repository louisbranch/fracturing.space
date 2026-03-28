package modules

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/testast"
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
			testast.AssertFuncDoesNotCallSelector(t, tc.modulePath, "Mount", tc.appPackage, tc.constructor)
			testast.AssertFuncCallsSelector(t, tc.composePath, tc.composeFunc, tc.appPackage, tc.constructor)
		})
	}
}

func TestSettingsCapabilitySplitStaysConstructorVisible(t *testing.T) {
	t.Parallel()

	assertStructFieldType(t, "settings/module.go", "Config", "Services", "handlerServices")
	assertStructFieldType(t, "settings/module.go", "Config", "Availability", "settingsSurfaceAvailability")
	testast.AssertFuncDoesNotCallSelector(t, "settings/module.go", "Mount", "settingsapp", "NewAccountService")
	testast.AssertFuncDoesNotCallSelector(t, "settings/module.go", "Mount", "settingsapp", "NewAIService")
	testast.AssertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsgateway", "NewAccountGateway")
	testast.AssertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsgateway", "NewAIGateway")
	testast.AssertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsapp", "NewAccountService")
	testast.AssertFuncCallsSelector(t, "settings/composition.go", "Compose", "settingsapp", "NewAIService")

	assertPackageHasType(t, "settings/app", "AccountServiceConfig")
	assertPackageHasType(t, "settings/app", "AIServiceConfig")
	assertPackageHasType(t, "settings/app", "accountService")
	assertPackageHasType(t, "settings/app", "aiService")
	assertPackageLacksType(t, "settings/app", "ServiceConfig")
	assertPackageHasFunc(t, "settings/app", "NewAccountService")
	assertPackageHasFunc(t, "settings/app", "NewAIService")
	assertPackageLacksFunc(t, "settings/app", "NewService")
}

func TestPublicAuthSurfaceServicesStaySplitEndToEnd(t *testing.T) {
	t.Parallel()

	assertStructFieldType(t, "publicauth/module.go", "Config", "PageService", "publicauthapp.PageService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "SessionService", "publicauthapp.SessionService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "PasskeyService", "publicauthapp.PasskeyService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "Recovery", "publicauthapp.RecoveryService")
	assertStructFieldType(t, "publicauth/module.go", "Config", "Principal", "principal.PrincipalResolver")
	testast.AssertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewPageService")
	testast.AssertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewSessionService")
	testast.AssertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewPasskeyService")
	testast.AssertFuncDoesNotCallSelector(t, "publicauth/module.go", "Mount", "publicauthapp", "NewRecoveryService")

	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Pages", "publicauthapp.PageService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Session", "publicauthapp.SessionService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Passkeys", "publicauthapp.PasskeyService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Recovery", "publicauthapp.RecoveryService")
	assertStructFieldType(t, "publicauth/handlers.go", "handlersConfig", "Principal", "principal.PrincipalResolver")
	testast.AssertFuncCallsSelector(t, "publicauth/handlers.go", "newHandlers", "publichandler", "NewBaseFromPrincipal")

	testast.AssertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewPageService")
	testast.AssertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewSessionService")
	testast.AssertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewPasskeyService")
	testast.AssertFuncCallsSelector(t, "publicauth/composition.go", "compose", "publicauthapp", "NewRecoveryService")
	testast.AssertFuncExists(t, "publicauth/composition.go", "ComposeSurfaceSet")

	assertPackageHasType(t, "publicauth/app", "pageService")
	assertPackageHasType(t, "publicauth/app", "sessionService")
	assertPackageHasType(t, "publicauth/app", "passkeyService")
	assertPackageHasType(t, "publicauth/app", "recoveryService")
	assertPackageHasFunc(t, "publicauth/app", "NewPageService")
	assertPackageHasFunc(t, "publicauth/app", "NewSessionService")
	assertPackageHasFunc(t, "publicauth/app", "NewPasskeyService")
	assertPackageHasFunc(t, "publicauth/app", "NewRecoveryService")
	assertPackageLacksFunc(t, "publicauth/app", "NewService")
	assertPackageLacksType(t, "publicauth/app", "Service")
	assertPackageLacksType(t, "publicauth/app", "Gateway")
}

func TestPublicAuthUsesSharedRedirectAndPrincipalSeams(t *testing.T) {
	t.Parallel()

	testast.AssertFileCallsSelector(t, "publicauth/handlers_pages.go", "redirectpath", "ResolveSafe")
	testast.AssertFileCallsSelector(t, "publicauth/app/service_helpers.go", "redirectpath", "ResolveSafe")
	testast.AssertFuncCallsSelector(t, "publicauth/handlers_session.go", "redirectAuthenticatedToApp", "h", "IsViewerSignedIn")
	assertStructFieldType(t, "publicauth/composition.go", "CompositionConfig", "Principal", "principal.PrincipalResolver")
	testast.AssertImportAbsent(t, "publicauth/handlers.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
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

	testast.AssertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "publicauth", "ComposeSurfaceSet")
	testast.AssertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "discovery", "Compose")
	testast.AssertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "profile", "Compose")
	testast.AssertFuncCallsSelector(t, "registry_public.go", "defaultPublicModules", "invite", "Compose")
	testast.AssertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "publicauthgateway", "NewGRPCGateway")
	testast.AssertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "profilegateway", "NewGRPCGateway")
	testast.AssertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "invitegateway", "NewGRPCGateway")
	testast.AssertFuncDoesNotCallSelector(t, "registry_public.go", "defaultPublicModules", "dashboardsync", "New")

	testast.AssertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "modulehandler", "NewBaseFromPrincipal")
	testast.AssertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "dashboard", "Compose")
	testast.AssertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "settings", "ComposeProtected")
	testast.AssertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "campaigns", "ComposeProtected")
	testast.AssertFuncCallsSelector(t, "registry_protected.go", "buildProtectedModules", "notifications", "Compose")
	testast.AssertFuncDoesNotCallSelector(t, "registry_protected.go", "buildProtectedModules", "dashboardgateway", "NewGRPCGateway")
	testast.AssertFuncDoesNotCallSelector(t, "registry_protected.go", "buildProtectedModules", "settingsgateway", "NewGRPCGateway")
	testast.AssertFuncDoesNotCallSelector(t, "registry_protected.go", "buildProtectedModules", "campaigngateway", "NewGRPCGateway")

	testast.AssertFuncCallsIdent(t, "registry.go", "Build", "newSharedServices")
	testast.AssertFuncDoesNotCallSelector(t, "registry.go", "Build", "dashboardsync", "New")
}

func TestRequestResolutionUsesPrincipalSeam(t *testing.T) {
	t.Parallel()

	testast.AssertTypeExists(t, "../principal/requeststate.go", "PageResolver")
	testast.AssertTypeExists(t, "../principal/requeststate.go", "PrincipalResolver")
	testast.AssertTypeExists(t, "../principal/callbacks.go", "ViewerFunc")
	testast.AssertTypeExists(t, "../principal/callbacks.go", "SignedInFunc")
	testast.AssertTypeExists(t, "../principal/callbacks.go", "UserIDFunc")
	testast.AssertTypeExists(t, "../principal/callbacks.go", "LanguageFunc")
	testast.AssertFuncExists(t, "../principal/requeststate.go", "NewBaseFromPageResolver")
	testast.AssertFuncExists(t, "../principal/requeststate.go", "ResolveLocalizedPage")
	testast.AssertTypeMissing(t, "../module/module.go", "ResolveViewer")
	testast.AssertTypeMissing(t, "../module/module.go", "ResolveSignedIn")
	testast.AssertTypeMissing(t, "../module/module.go", "ResolveUserID")
	testast.AssertTypeMissing(t, "../module/module.go", "ResolveLanguage")
	testast.AssertImportPresent(t, "../platform/modulehandler/modulehandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	testast.AssertImportPresent(t, "../platform/publichandler/publichandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	testast.AssertImportPresent(t, "../platform/pagerender/pagerender.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	testast.AssertImportPresent(t, "../platform/weberror/weberror.go", "github.com/louisbranch/fracturing.space/internal/services/web/principal")
	testast.AssertImportAbsent(t, "../platform/modulehandler/modulehandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
	testast.AssertImportAbsent(t, "../platform/publichandler/publichandler.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
	testast.AssertImportAbsent(t, "../platform/pagerender/pagerender.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
	testast.AssertImportAbsent(t, "../platform/weberror/weberror.go", "github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver")
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

	parsed := testast.ParseFile(t, path)
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

func assertPackageHasType(t *testing.T, dir, typeName string) {
	t.Helper()

	if packageHasType(t, dir, typeName) {
		return
	}
	t.Fatalf("%s missing type %s", dir, typeName)
}

func assertPackageLacksType(t *testing.T, dir, typeName string) {
	t.Helper()

	if packageHasType(t, dir, typeName) {
		t.Fatalf("%s unexpectedly defines type %s", dir, typeName)
	}
}

func assertPackageHasFunc(t *testing.T, dir, funcName string) {
	t.Helper()

	if packageHasFunc(t, dir, funcName) {
		return
	}
	t.Fatalf("%s missing func %s", dir, funcName)
}

func assertPackageLacksFunc(t *testing.T, dir, funcName string) {
	t.Helper()

	if packageHasFunc(t, dir, funcName) {
		t.Fatalf("%s unexpectedly defines func %s", dir, funcName)
	}
}

func packageHasType(t *testing.T, dir, typeName string) bool {
	t.Helper()

	for _, path := range goFilesUnder(t, dir, false) {
		parsed := testast.ParseFile(t, path)
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
	}
	return false
}

func packageHasFunc(t *testing.T, dir, funcName string) bool {
	t.Helper()

	for _, path := range goFilesUnder(t, dir, false) {
		parsed := testast.ParseFile(t, path)
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Name != nil && fn.Name.Name == funcName {
				return true
			}
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
