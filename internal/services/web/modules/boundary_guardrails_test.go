package modules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestProtectedModuleCompatibilityShimsAreRemoved(t *testing.T) {
	t.Parallel()

	paths := []string{
		"campaigns/contracts.go",
		"campaigns/contracts_gateway.go",
		"campaigns/handlers_detail_pages.go",
		"campaigns/routes_surface_core.go",
		"campaigns/routes_surface_workflow.go",
		"campaigns/routes_surface_mutations.go",
		"campaigns/service_factory.go",
		"campaigns/workflow_contract.go",
		"settings/contracts.go",
		"settings/service.go",
		"settings/gateway_grpc.go",
		"settings/gateway_unavailable.go",
		"notifications/contracts.go",
		"notifications/service.go",
		"notifications/gateway_grpc.go",
		"notifications/gateway_unavailable.go",
		"dashboard/contracts.go",
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
		{path: "campaigns/module.go", pkg: "campaignapp", method: "NewService"},
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

func TestSettingsModuleWiresAccountAndAIAppServicesDirectly(t *testing.T) {
	t.Parallel()

	assertMountCallsSelector(t, "settings/module.go", "settingsapp", "NewAccountService")
	assertMountCallsSelector(t, "settings/module.go", "settingsapp", "NewAIService")
	assertFileDoesNotContain(t, "settings/module.go", "settingsapp.NewService(m.gateway)")
	assertFileDoesNotContain(t, "settings/module.go", "newHandlers(svc, svc, svc, svc, svc")
	assertFileContains(t, "settings/app/service.go", "type AccountServiceConfig struct")
	assertFileContains(t, "settings/app/service.go", "type AIServiceConfig struct")
}

func TestProfileCompatibilityShimsAreRemoved(t *testing.T) {
	t.Parallel()

	paths := []string{
		"profile/contracts.go",
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

func TestPublicAuthSurfaceWrapperPackagesAreRemoved(t *testing.T) {
	t.Parallel()

	paths := []string{
		"publicauth/surfaces/shell/module.go",
		"publicauth/surfaces/passkeys/module.go",
		"publicauth/surfaces/authredirect/module.go",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			_, err := os.Stat(path)
			if err == nil {
				t.Fatalf("%q exists; publicauth route-surface ownership should stay in the root package", path)
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

func TestPublicAuthUsesSharedRedirectPathSanitizer(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "publicauth/handlers_pages.go", "redirectpath.ResolveSafe(")
	assertFileContains(t, "publicauth/app/service.go", "redirectpath.ResolveSafe(")
	assertFileDoesNotContain(t, "publicauth/handlers_session.go", "func resolveSafeRedirectPath(")
	assertFileContains(t, "publicauth/module.go", "ResolveSignedIn module.ResolveSignedIn")
	assertFileContains(t, "publicauth/handlers_session.go", "h.IsViewerSignedIn(r)")
	assertFileDoesNotContain(t, "publicauth/app/types.go", "HasValidWebSession")
	assertFileDoesNotContain(t, "publicauth/handlers_session.go", "HasValidWebSession(")
	assertFileDoesNotContain(t, "publicauth/gateway/grpc.go", "GetWebSession")
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
	assertRegistryGatewayCallUsesNestedDepField(t, "registry_public.go", "profilegateway", "NewGRPCGateway", 0, "Profile", "AuthClient")
	assertRegistryGatewayCallUsesNestedDepField(t, "registry_public.go", "profilegateway", "NewGRPCGateway", 1, "Profile", "SocialClient")
	assertFileContains(t, "registry_protected.go", "dashboard.Compose(dashboard.CompositionConfig{")
	assertFileContains(t, "registry_protected.go", "notifications.Compose(notifications.CompositionConfig{")
	assertFileContains(t, "registry_protected.go", "settings.Compose(settings.CompositionConfig{")
	assertFileContains(t, "registry_protected.go", "campaigns.Compose(campaigns.CompositionConfig{")
	assertFileDoesNotContain(t, "registry_protected.go", "dashboardgateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "notificationsgateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "settingsgateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigngateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigns.GameSystem")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigns.CharacterCreationWorkflow")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigns.CampaignGateway")
}

func TestCampaignDetailRenderSeamIsAreaOwned(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/handlers_detail_scaffold.go", "campaignrender.Fragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_scaffold.go", "webtemplates.CampaignDetailFragment(")
	assertFileContains(t, "campaigns/handlers_detail_context.go", "campaignrender.DetailView")
	assertFileContains(t, "campaigns/view_participants.go", "campaignrender.ParticipantView")
	assertFileContains(t, "campaigns/view_sessions.go", "campaignrender.SessionView")
}

func TestCampaignListCreateTemplatesAreAreaOwned(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/handlers_list_create.go", "CampaignListFragment(")
	assertFileContains(t, "campaigns/handlers_list_create.go", "CampaignStartFragment(")
	assertFileContains(t, "campaigns/handlers_list_create.go", "CampaignCreateFragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_list_create.go", "webtemplates.CampaignListFragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_list_create.go", "webtemplates.CampaignStartFragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_list_create.go", "webtemplates.CampaignCreateFragment(")
	assertFileContains(t, "campaigns/page.templ", "type CampaignListItem struct")
	assertFileContains(t, "campaigns/page.templ", "type CampaignCreateFormValues struct")
	assertFileContains(t, "campaigns/page.templ", "templ CampaignListFragment(")
	assertFileMissing(t, "../templates/campaigns.templ")
}

func TestCampaignChatAndCreationTemplatesAreAreaOwned(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/handlers_chat.go", "CampaignChatPage(")
	assertFileDoesNotContain(t, "campaigns/handlers_chat.go", "webtemplates.CampaignChatPage(")
	assertFileContains(t, "campaigns/chat_page.templ", "templ CampaignChatPage(")
	assertFileContains(t, "campaigns/handlers_creation_page.go", "campaignrender.CharacterCreationPage(")
	assertFileDoesNotContain(t, "campaigns/handlers_creation_page.go", "webtemplates.CharacterCreationPage(")
	assertFileContains(t, "campaigns/render/character_creation.templ", "templ CharacterCreationPage(")
	assertFileMissing(t, "../templates/campaigns_chat.templ")
	assertFileMissing(t, "../templates/character_creation.templ")
}

func TestCampaignServiceConstructorUsesExplicitCapabilityConfig(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/module.go", "campaignapp.ServiceConfig{")
	assertFileContains(t, "campaigns/module.go", "readGateway")
	assertFileContains(t, "campaigns/module.go", "mutationGateway")
	assertFileContains(t, "campaigns/module.go", "authzGateway")
	assertFileContains(t, "campaigns/module.go", "ReadGateway:")
	assertFileContains(t, "campaigns/module.go", "MutationGateway:")
	assertFileContains(t, "campaigns/module.go", "AuthzGateway:")
	assertFileDoesNotContain(t, "campaigns/module.go", "m.gateway")
	assertFileDoesNotContain(t, "campaigns/module.go", "Gateway          campaignapp.CampaignGateway")
	assertFileDoesNotContain(t, "campaigns/module.go", "campaignapp.NewService(m.gateway)")
	assertFileContains(t, "campaigns/composition.go", "ReadGateway:      gateway")
	assertFileContains(t, "campaigns/composition.go", "MutationGateway:  gateway")
	assertFileContains(t, "campaigns/composition.go", "AuthzGateway:     gateway")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type ServiceConfig struct")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "func NewService(gateway CampaignGateway)")
}

func TestCampaignCharacterCreationWorkflowOwnershipStaysOutOfApp(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/workflow/service.go", "type AppService interface")
	assertFileContains(t, "campaigns/workflow/service.go", "func (s Service) LoadPage(")
	assertFileContains(t, "campaigns/workflow/service.go", "func (s Service) ApplyStep(")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "CampaignCharacterCreation(context.Context, string, string, language.Tag")
	assertFileDoesNotContain(t, "campaigns/handlers_workflow.go", "workflow.ParseStepInput(")
	assertFileDoesNotContain(t, "campaigns/handlers_creation_page.go", "CampaignCharacterCreation(")
	assertFileMissing(t, "campaigns/app/workflow.go")
}

func TestSelectedModuleRootsDoNotUseAliasWalls(t *testing.T) {
	t.Parallel()

	assertFileDoesNotContain(t, "dashboard/handlers.go", "type dashboardService =")
	assertFileDoesNotContain(t, "notifications/handlers.go", "type notificationService =")
	assertFileDoesNotContain(t, "profile/handlers.go", "type profileService =")
	assertFileDoesNotContain(t, "settings/handlers.go", "type settingsProfileService =")
	assertFileDoesNotContain(t, "settings/handlers.go", "type settingsLocaleService =")
	assertFileDoesNotContain(t, "settings/handlers.go", "type settingsSecurityService =")
	assertFileDoesNotContain(t, "settings/handlers.go", "type settingsAIKeyService =")
	assertFileDoesNotContain(t, "settings/handlers.go", "type settingsAIAgentService =")
	assertFileDoesNotContain(t, "discovery/gateway.go", "type StarterEntry =")
	assertFileDoesNotContain(t, "discovery/gateway.go", "type Gateway =")
}

func TestModulesPackageDoesNotReexportModuleContractAliases(t *testing.T) {
	t.Parallel()

	parsed := parseFile(t, "module.go")
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil {
				continue
			}
			if typeSpec.Name.Name != "Module" && typeSpec.Name.Name != "Mount" {
				continue
			}
			if _, isAlias := typeSpec.Type.(*ast.Ident); isAlias && typeSpec.Assign.IsValid() {
				t.Fatalf("modules/module.go reexports %s alias; singular internal/services/web/module should stay the only module contract owner", typeSpec.Name.Name)
			}
		}
	}
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

func assertFileContains(t *testing.T, path, substring string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if !strings.Contains(string(content), substring) {
		t.Fatalf("%s does not contain %q", path, substring)
	}
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

func assertRegistryUsesGatewayDepsLiteral(t *testing.T, path, pkgName, methodName string) {
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
		if !ok || ident.Name != pkgName || len(call.Args) != 1 {
			return true
		}
		if _, ok := call.Args[0].(*ast.CompositeLit); !ok {
			return true
		}
		found = true
		return false
	})
	if !found {
		t.Fatalf("%s does not call %s.%s with an explicit deps literal", path, pkgName, methodName)
	}
}

func assertFileDoesNotContain(t *testing.T, path, fragment string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if len(data) == 0 {
		t.Fatalf("%s unexpectedly empty", path)
	}
	if strings.Contains(string(data), fragment) {
		t.Fatalf("%s still contains %q", path, fragment)
	}
}

func assertFileMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("%s exists; expected legacy file to be removed", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}
