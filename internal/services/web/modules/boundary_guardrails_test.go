package modules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestProtectedModuleRootsWireAppServicesDirectly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path   string
		pkg    string
		method string
	}{
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
	assertFileContains(t, "settings/app/service.go", "type serviceConfig struct")
	assertFileDoesNotContain(t, "settings/app/service.go", "func newService(gateway Gateway)")
	assertFileDoesNotContain(t, "settings/app/service.go", "type ServiceConfig struct")
	assertFileDoesNotContain(t, "settings/app/service.go", "func NewService(config")
	assertFileDoesNotContain(t, "settings/app/contracts.go", "type Service interface")
	assertFileDoesNotContain(t, "settings/app/contracts.go", "type Gateway interface")
	assertFileContains(t, "settings/app/service_test_helpers_test.go", "type testGateway interface")
	assertFileContains(t, "settings/app/service_test_helpers_test.go", "func newService(gateway testGateway) service")
}

func TestProfileRootWiresAppServiceDirectly(t *testing.T) {
	t.Parallel()
	assertMountCallsSelector(t, "profile/module.go", "profileapp", "NewService")
}

func TestPublicAuthRootWiresAppServiceDirectly(t *testing.T) {
	t.Parallel()
	assertFileContains(t, "publicauth/module.go", "services       handlerServices")
	assertFileContains(t, "publicauth/module.go", "Services:  m.services")
	assertFileDoesNotContain(t, "publicauth/module.go", "publicauthapp.NewService(")
	assertFileDoesNotContain(t, "publicauth/module.go", "publicauthapp.NewPageService(")
	assertFileDoesNotContain(t, "publicauth/module.go", "publicauthapp.NewSessionService(")
	assertFileDoesNotContain(t, "publicauth/module.go", "publicauthapp.NewPasskeyService(")
	assertFileDoesNotContain(t, "publicauth/module.go", "publicauthapp.NewRecoveryService(")
	assertFileContains(t, "publicauth/module.go", "services       handlerServices")
	assertFileDoesNotContain(t, "publicauth/module.go", "gateway        publicauthapp.Gateway")
	assertFileDoesNotContain(t, "publicauth/module.go", "authBaseURL")
	assertFileDoesNotContain(t, "publicauth/handlers.go", "func newHandlersFromService(")
	assertFileContains(t, "publicauth/handlers.go", "type gatewayServices interface")
	assertFileContains(t, "publicauth/handlers.go", "func newHandlerServicesFromGateway(gateway gatewayServices, authBaseURL string) handlerServices")
	assertFileContains(t, "publicauth/handlers.go", "func normalizeHandlerServices(services handlerServices) handlerServices")
	assertFileDoesNotContain(t, "publicauth/app/types.go", "type Service interface")
	assertFileDoesNotContain(t, "publicauth/app/types.go", "type Gateway interface")
	assertFileDoesNotContain(t, "publicauth/app/service.go", "func NewService(")
	assertFileContains(t, "publicauth/app/service.go", "type serviceConfig struct")
	assertFileContains(t, "publicauth/app/service.go", "func NewPageService(authBaseURL string) PageService")
	assertFileContains(t, "publicauth/app/service.go", "func NewSessionService(gateway SessionGateway, authBaseURL string) SessionService")
	assertFileContains(t, "publicauth/app/service.go", "func NewPasskeyService(gateway PasskeyGateway, authBaseURL string) PasskeyService")
	assertFileContains(t, "publicauth/app/service.go", "func NewRecoveryService(gateway RecoveryGateway, authBaseURL string) RecoveryService")
	assertFileContains(t, "publicauth/handlers_test_helpers_test.go", "func newHandlersFromGateway(")
	assertFileContains(t, "publicauth/app/service_test_helpers_test.go", "type testGateway interface")
	assertFileContains(t, "publicauth/app/service_test_helpers_test.go", "func newService(gateway testGateway, authBaseURL string) service")
	assertFileContains(t, "publicauth/composition.go", "Services:    newHandlerServicesFromGateway(")
}

func TestPublicAuthUsesSharedRedirectPathSanitizer(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "publicauth/handlers_pages.go", "redirectpath.ResolveSafe(")
	assertFileContains(t, "publicauth/app/service.go", "redirectpath.ResolveSafe(")
	assertFileDoesNotContain(t, "publicauth/handlers_session.go", "func resolveSafeRedirectPath(")
	assertFileContains(t, "publicauth/module.go", "principal      requestresolver.PrincipalResolver")
	assertFileDoesNotContain(t, "publicauth/module.go", "ResolveSignedIn module.ResolveSignedIn")
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

	assertFileDoesNotContain(t, "registry.go", "deps.SocialClient")
	assertFileDoesNotContain(t, "registry_public.go", "deps.SocialClient")
	assertFileDoesNotContain(t, "registry_protected.go", "deps.SocialClient")
	assertFileContains(t, "profile/composition.go", "profilegateway.NewGRPCGateway(config.AuthClient, config.SocialClient)")
	assertFileContains(t, "registry_protected.go", "dashboard.Compose(dashboard.CompositionConfig{")
	assertFileContains(t, "registry_protected.go", "notifications.Compose(notifications.CompositionConfig{")
	assertFileContains(t, "registry_protected.go", "settings.Compose(settings.CompositionConfig{")
	assertFileContains(t, "registry_protected.go", "campaigns.ComposeProtected(campaigns.ProtectedSurfaceOptions{")
	assertFileDoesNotContain(t, "registry_protected.go", "dashboardgateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "notificationsgateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "settingsgateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigngateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigns.GameSystem")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigns.CharacterCreationWorkflow")
	assertFileDoesNotContain(t, "registry_protected.go", "campaigns.CampaignGateway")
	assertFileDoesNotContain(t, "registry_protected.go", "campaignsConfigured(")
	assertFileContains(t, "campaigns/composition.go", "func ComposeProtected(options ProtectedSurfaceOptions, deps Dependencies) (module.Module, bool)")
}

func TestPublicRegistryUsesAreaOwnedCompositionEntrypoints(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "registry_public.go", "publicauth.ComposeSurfaceSet(publicauth.SurfaceSetConfig{")
	assertFileContains(t, "registry_public.go", "discovery.Compose(discovery.CompositionConfig{")
	assertFileContains(t, "registry_public.go", "profile.Compose(profile.CompositionConfig{")
	assertFileContains(t, "registry_public.go", "invite.ComposePublic(invite.PublicSurfaceOptions{")
	assertFileDoesNotContain(t, "registry_public.go", "publicauth.New(publicauth.Config{")
	assertFileDoesNotContain(t, "registry_public.go", "SurfaceShell")
	assertFileDoesNotContain(t, "registry_public.go", "SurfacePasskeys")
	assertFileDoesNotContain(t, "registry_public.go", "SurfaceAuthRedirect")
	assertFileDoesNotContain(t, "registry_public.go", "publicauthgateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_public.go", "discovery.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_public.go", "profilegateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_public.go", "invitegateway.NewGRPCGateway")
	assertFileDoesNotContain(t, "registry_public.go", "publichandler.NewBase")
	assertFileDoesNotContain(t, "registry_public.go", "dashboardsync.New")
	assertFileDoesNotContain(t, "registry_public.go", "deps.Campaigns.InviteClient")
	assertFileDoesNotContain(t, "registry_public.go", "deps.Campaigns.AuthClient")
	assertFileDoesNotContain(t, "registry_public.go", "deps.DashboardSync.UserHubControlClient")
	assertFileDoesNotContain(t, "registry_public.go", "deps.DashboardSync.GameEventClient")
	assertFileContains(t, "publicauth/composition.go", "func Compose(config CompositionConfig) module.Module")
	assertFileContains(t, "publicauth/composition.go", "func ComposeSurfaceSet(config SurfaceSetConfig) []module.Module")
	assertFileContains(t, "registry_public.go", "Principal:   principal")
	assertFileContains(t, "registry_public.go", "Principal:    principal")
	assertFileContains(t, "discovery/composition.go", "func Compose(config CompositionConfig) module.Module")
	assertFileContains(t, "profile/composition.go", "func Compose(config CompositionConfig) module.Module")
	assertFileContains(t, "invite/composition.go", "func Compose(config CompositionConfig) module.Module")
	assertFileContains(t, "invite/composition.go", "func ComposePublic(options PublicSurfaceOptions, deps Dependencies) module.Module")
	assertFileContains(t, "profile/composition.go", "Principal    requestresolver.PrincipalResolver")
	assertFileContains(t, "invite/composition.go", "Principal   requestresolver.PrincipalResolver")
	assertFileContains(t, "publicauth/composition.go", "Principal   requestresolver.PrincipalResolver")
	assertFileContains(t, "profile/module.go", "publichandler.NewBaseFromPrincipal(m.principal)")
	assertFileContains(t, "invite/module.go", "principal   requestresolver.PrincipalResolver")
	assertFileContains(t, "invite/handlers.go", "publichandler.NewBaseFromPrincipal(principal)")
	assertFileContains(t, "invite/handlers.go", "h.RequestUserID(r)")
	assertFileContains(t, "publicauth/handlers.go", "publichandler.NewBaseFromPrincipal(config.Principal)")
}

func TestModuleDependencyBindingDelegatesToOwningAreas(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "dependencies.go", "publicauth.BindAuthDependency(&deps.PublicAuth, conn)")
	assertFileContains(t, "dependencies.go", "profile.BindAuthDependency(&deps.Profile, conn)")
	assertFileContains(t, "dependencies.go", "profile.BindSocialDependency(&deps.Profile, conn)")
	assertFileContains(t, "dependencies.go", "settings.BindAuthDependency(&deps.Settings, conn)")
	assertFileContains(t, "dependencies.go", "settings.BindSocialDependency(&deps.Settings, conn)")
	assertFileContains(t, "dependencies.go", "settings.BindAIDependency(&deps.Settings, conn)")
	assertFileContains(t, "dependencies.go", "campaigns.BindAuthDependency(&deps.Campaigns, conn)")
	assertFileContains(t, "dependencies.go", "campaigns.BindSocialDependency(&deps.Campaigns, conn)")
	assertFileContains(t, "dependencies.go", "campaigns.BindGameDependency(&deps.Campaigns, conn)")
	assertFileContains(t, "dependencies.go", "campaigns.BindAIDependency(&deps.Campaigns, conn)")
	assertFileContains(t, "dependencies.go", "invite.BindAuthDependency(&deps.Invite, conn)")
	assertFileContains(t, "dependencies.go", "invite.BindGameDependency(&deps.Invite, conn)")
	assertFileContains(t, "dependencies.go", "invite.BindUserHubDependency(&deps.Invite, conn)")
	assertFileContains(t, "dependencies.go", "discovery.BindDependency(&deps.Discovery, conn)")
	assertFileContains(t, "dependencies.go", "notifications.BindDependency(&deps.Notifications, conn)")
	assertFileContains(t, "dependencies.go", "dashboard.BindUserHubDependency(&deps.Dashboard, conn)")
	assertFileContains(t, "dependencies.go", "dashboard.BindStatusDependency(&deps.Dashboard, conn)")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Campaigns.CampaignClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Campaigns.AgentClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Campaigns.SocialClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Profile.AuthClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Profile.SocialClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Settings.AccountClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Settings.CredentialClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.PublicAuth.AuthClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Discovery.DiscoveryClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Notifications.NotificationClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Dashboard.UserHubClient =")
	assertFileDoesNotContain(t, "dependencies.go", "deps.Dashboard.StatusClient =")
	assertFileContains(t, "dashboard/dependencies.go", "func BindUserHubDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "dashboard/dependencies.go", "func BindStatusDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "profile/dependencies.go", "func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "profile/dependencies.go", "func BindSocialDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "settings/dependencies.go", "func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "settings/dependencies.go", "func BindSocialDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "settings/dependencies.go", "func BindAIDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "publicauth/dependencies.go", "func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "discovery/dependencies.go", "func BindDependency(deps *Dependencies, conn *grpc.ClientConn)")
	assertFileContains(t, "notifications/dependencies.go", "func BindDependency(deps *Dependencies, conn *grpc.ClientConn)")
}

func TestWebModulePlaybookMatchesCurrentPublicModuleContract(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "../../../../docs/guides/web-module-playbook.md", "publichandler.Base")
	assertFileContains(t, "../../../../docs/guides/web-module-playbook.md", "requestresolver.PrincipalResolver")
	assertFileDoesNotContain(t, "../../../../docs/guides/web-module-playbook.md", "viewer/user-id/language resolvers are not injected")
}

func TestCampaignDetailRenderSeamIsAreaOwned(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/handlers_detail_scaffold.go", "body templ.Component")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_scaffold.go", "campaignrender.Fragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_scaffold.go", "webtemplates.CampaignDetailFragment(")
	assertFileMissing(t, "campaigns/handlers_detail_context.go")
	assertFileContains(t, "campaigns/detail_views_shared.go", "func (p *campaignPageContext) baseDetailView(")
	assertFileContains(t, "campaigns/detail_views_shared.go", "campaignrender.CampaignDetailBaseView")
	assertFileDoesNotContain(t, "campaigns/detail_views_shared.go", "ParticipantsPageView")
	assertFileDoesNotContain(t, "campaigns/detail_views_shared.go", "CharacterDetailPageView")
	assertFileContains(t, "campaigns/campaign_page_context.go", "type campaignPageContext struct")
	assertFileContains(t, "campaigns/campaign_page_context.go", "func (h handlers) loadCampaignPage(")
	assertFileContains(t, "campaigns/handlers_route_params.go", "func (h handlers) withCampaignID(")
	assertFileContains(t, "campaigns/handlers_route_params.go", "func (h handlers) withCampaignAndCharacterID(")
	assertFileContains(t, "campaigns/detail_views_overview.go", "func (p *campaignPageContext) overviewView(")
	assertFileContains(t, "campaigns/detail_views_participants.go", "func (p *campaignPageContext) participantsView(")
	assertFileContains(t, "campaigns/detail_views_characters.go", "func (p *campaignPageContext) characterDetailView(")
	assertFileContains(t, "campaigns/detail_views_sessions_invites.go", "func (p *campaignPageContext) sessionsView(")
	assertFileContains(t, "campaigns/detail_views_overview.go", "campaignrender.OverviewPageView")
	assertFileContains(t, "campaigns/detail_views_participants.go", "campaignrender.ParticipantsPageView")
	assertFileContains(t, "campaigns/detail_views_characters.go", "campaignrender.CharacterDetailPageView")
	assertFileContains(t, "campaigns/detail_views_sessions_invites.go", "campaignrender.SessionDetailPageView")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_overview.go", "page.baseDetailView(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_participants.go", "page.baseDetailView(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_characters.go", "page.baseDetailView(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_sessions_invites.go", "page.baseDetailView(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_overview.go", "campaignrender.Fragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_participants.go", "campaignrender.Fragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_characters.go", "campaignrender.Fragment(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_sessions_invites.go", "campaignrender.Fragment(")
	assertFileDoesNotContain(t, "campaigns/detail_views_overview.go", "campaignrender.DetailView")
	assertFileDoesNotContain(t, "campaigns/detail_views_participants.go", "campaignrender.DetailView")
	assertFileDoesNotContain(t, "campaigns/detail_views_characters.go", "campaignrender.DetailView")
	assertFileDoesNotContain(t, "campaigns/detail_views_sessions_invites.go", "campaignrender.DetailView")
	assertFileContains(t, "campaigns/render/detail_overview.go", "return overviewFragment(")
	assertFileContains(t, "campaigns/render/detail_participants.go", "return participantsFragment(")
	assertFileContains(t, "campaigns/render/detail_characters.go", "return characterDetailFragment(")
	assertFileContains(t, "campaigns/render/detail_sessions.go", "return sessionsFragment(")
	assertFileContains(t, "campaigns/render/detail_invites.go", "return invitesFragment(")
	assertFileDoesNotContain(t, "campaigns/render/detail.go", "Marker")
	assertFileDoesNotContain(t, "campaigns/render/detail.go", "type detailPageView struct")
	assertFileContains(t, "campaigns/render/detail.go", "type CampaignDetailBaseView struct")
	assertFileDoesNotContain(t, "campaigns/render/detail.go", "type OverviewPageView struct")
	assertFileDoesNotContain(t, "campaigns/render/detail.go", "type ParticipantView struct")
	assertFileDoesNotContain(t, "campaigns/render/detail.go", "type CharacterView struct")
	assertFileDoesNotContain(t, "campaigns/render/detail.go", "type SessionView struct")
	assertFileDoesNotContain(t, "campaigns/render/detail.go", "type InviteView struct")
	assertFileMissing(t, "campaigns/render/detail.templ")
	assertFileContains(t, "campaigns/render/detail_overview.templ", "templ overviewFragment(view OverviewPageView")
	assertFileContains(t, "campaigns/render/detail_participants.templ", "templ participantsFragment(view ParticipantsPageView")
	assertFileContains(t, "campaigns/render/detail_characters.templ", "templ characterDetailFragment(view CharacterDetailPageView")
	assertFileContains(t, "campaigns/render/detail_sessions.templ", "templ sessionsFragment(view SessionsPageView")
	assertFileContains(t, "campaigns/render/detail_invites.templ", "templ invitesFragment(view InvitesPageView")
	assertFileContains(t, "campaigns/render/detail_shared.templ", "templ campaignCharacterEditorForm(editor CharacterEditorView")
	assertFileMissing(t, "campaigns/render/helpers.go")
	assertFileContains(t, "campaigns/render/helpers_overview.go", "func campaignOverviewTheme(")
	assertFileContains(t, "campaigns/render/helpers_participants.go", "func campaignParticipantEditURL(")
	assertFileContains(t, "campaigns/render/helpers_characters.go", "func campaignCharacterDetailURL(")
	assertFileContains(t, "campaigns/render/helpers_sessions.go", "func campaignSessionByID(")
	assertFileContains(t, "campaigns/render/helpers_invites.go", "func campaignInviteStatusLabel(")
	assertFileContains(t, "campaigns/view_participants.go", "campaignrender.ParticipantView")
	assertFileContains(t, "campaigns/view_sessions.go", "campaignrender.SessionView")
	assertFileContains(t, "campaigns/composition.go", "newServiceConfigsFromGRPCDeps(newGatewayDeps(config), config.AssetBaseURL)")
	assertFileContains(t, "campaigns/composition_service_config_catalog.go", "campaigngateway.NewCatalogReadGateway(")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "campaigngateway.NewParticipantReadGateway(")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "campaigngateway.NewCharacterReadGateway(")
	assertFileContains(t, "campaigns/composition_service_config.go", "campaigngateway.NewAuthorizationGateway(")
	assertFileDoesNotContain(t, "campaigns/composition.go", "campaigngateway.NewGRPCGateway(")
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
}

func TestCampaignCreationTemplatesAreAreaOwned(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/handlers_creation_page.go", "campaignrender.CharacterCreationPage(")
	assertFileDoesNotContain(t, "campaigns/handlers_creation_page.go", "webtemplates.CharacterCreationPage(")
	assertFileContains(t, "campaigns/render/character_creation.templ", "templ CharacterCreationPage(")
}

func TestCampaignServiceConstructorUsesExplicitCapabilityConfig(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/module.go", "services         handlerServices")
	assertFileContains(t, "campaigns/module.go", "mountErr         error")
	assertFileContains(t, "campaigns/module.go", "mountErr:         validateHandlerServices(config.Services)")
	assertFileContains(t, "campaigns/module.go", "func validateHandlerServices(services handlerServices) error")
	assertFileDoesNotContain(t, "campaigns/module.go", "m.gateway")
	assertFileContains(t, "campaigns/handlers.go", "func newHandlerServices(config serviceConfigs)")
	assertFileContains(t, "campaigns/handlers.go", "type campaignPageHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type catalogHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type starterHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type overviewHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type participantHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type characterHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type creationHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type sessionHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "type inviteHandlerServices struct")
	assertFileContains(t, "campaigns/handlers.go", "pages            campaignPageHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "catalog          catalogHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "overview         overviewHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "participants     participantHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "characters       characterHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "creation         creationHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "sessions         sessionHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "invites          inviteHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "Page         campaignPageHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "Catalog      catalogHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "Overview     overviewHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "Participants participantHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "Characters   characterHandlerServices")
	assertFileContains(t, "campaigns/handlers.go", "Creation     campaignCreationAppServices")
	assertFileContains(t, "campaigns/handlers.go", "Invites      inviteHandlerServices")
	assertFileDoesNotContain(t, "campaigns/handlers.go", "game              campaignapp.CampaignGameService")
	assertFileDoesNotContain(t, "campaigns/handlers.go", "Game               campaignapp.CampaignGameService")
	assertFileDoesNotContain(t, "campaigns/handlers.go", "participantReads  campaignapp.CampaignParticipantReadService")
	assertFileDoesNotContain(t, "campaigns/handlers.go", "participantMutate campaignapp.CampaignParticipantMutationService")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewParticipantReadService(config.Participants.Read, config.Participants.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewParticipantMutationService(config.Participants.Mutation, config.Participants.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewAutomationReadService(config.Overview.AutomationRead, config.Overview.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewAutomationMutationService(config.Overview.AutomationMutation, config.Overview.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewCharacterReadService(config.Characters.Read, config.Characters.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewCharacterControlService(config.Characters.Control, config.Characters.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewCharacterMutationService(config.Characters.Mutation, config.Characters.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewSessionReadService(config.Page.SessionRead)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewSessionMutationService(config.Sessions.Mutation, config.Page.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewInviteReadService(config.Invites.Read, config.Invites.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "campaignapp.NewInviteMutationService(config.Invites.Mutation, config.Invites.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "workspace := campaignapp.NewWorkspaceService(config.Page.Workspace)")
	assertFileContains(t, "campaigns/handlers.go", "authorization := campaignapp.NewAuthorizationService(config.Page.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "participantReads := campaignapp.NewParticipantReadService(config.Participants.Read, config.Participants.Authorization)")
	assertFileContains(t, "campaigns/handlers.go", "inviteParticipantReads := campaignapp.NewParticipantReadService(config.Invites.ParticipantRead, config.Invites.Authorization)")
	assertFileDoesNotContain(t, "campaigns/handlers.go", "campaignapp.NewGameService(config.Game)")
	assertFileDoesNotContain(t, "campaigns/handlers.go", "func newHandlersFromService(")
	assertFileContains(t, "campaigns/campaign_page_context.go", "h.pages.workspace.CampaignWorkspace(")
	assertFileContains(t, "campaigns/campaign_page_context.go", "h.pages.sessionReads.CampaignSessions(")
	assertFileContains(t, "campaigns/campaign_page_context.go", "h.pages.authorization.RequireManageInvites(")
	assertFileContains(t, "campaigns/handlers_detail_overview.go", "h.overview.automationReads.CampaignAIBindingSummary(")
	assertFileContains(t, "campaigns/handlers_detail_participants.go", "h.participants.reads.CampaignParticipants(")
	assertFileContains(t, "campaigns/handlers_detail_characters.go", "h.characters.reads.CampaignCharacters(")
	assertFileContains(t, "campaigns/handlers_detail_sessions_invites.go", "h.invites.reads.CampaignInvites(")
	assertFileContains(t, "campaigns/handlers_detail_sessions_invites.go", "h.invites.participantReads.CampaignParticipants(")
	assertFileContains(t, "campaigns/handlers_mutation.go", "h.sessions.mutation.StartSession(")
	assertFileContains(t, "campaigns/handlers_mutation.go", "h.characters.mutation.CreateCharacter(")
	assertFileContains(t, "campaigns/handlers_mutation.go", "h.invites.mutation.CreateInvite(")
	assertFileContains(t, "campaigns/handlers_workflow.go", "h.creation.mutation.ApplyStep(")
	assertFileContains(t, "campaigns/handlers_creation_page.go", "h.creation.pages.LoadPage(")
	assertFileContains(t, "campaigns/routes_starters.go", "h.starters.starters == nil")
	assertFileContains(t, "campaigns/composition.go", "serviceConfigs := newServiceConfigsFromGRPCDeps(newGatewayDeps(config), config.AssetBaseURL)")
	assertFileContains(t, "campaigns/composition.go", "Services:         newHandlerServices(serviceConfigs)")
	assertFileDoesNotContain(t, "campaigns/composition.go", "InteractionClient")
	assertFileMissing(t, "campaigns/composition_services.go")
	assertFileContains(t, "campaigns/composition_gateway_deps.go", "func newGatewayDeps(config CompositionConfig) campaigngateway.GRPCGatewayDeps")
	assertFileDoesNotContain(t, "campaigns/composition_gateway_deps.go", "GameRead:")
	assertFileContains(t, "campaigns/composition_service_config.go", "func newServiceConfigsFromGRPCDeps(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) serviceConfigs")
	assertFileContains(t, "campaigns/composition_service_config_catalog.go", "func newCatalogServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CatalogServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "func newParticipantReadServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.ParticipantReadServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "func newParticipantMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.ParticipantMutationServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "func newCharacterReadServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CharacterReadServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "func newAutomationReadServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.AutomationReadServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "func newAutomationMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.AutomationMutationServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "func newCharacterControlServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CharacterControlServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_people.go", "func newCharacterMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.CharacterMutationServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_sessions.go", "func newSessionReadServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.SessionReadServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_sessions.go", "func newSessionMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.SessionMutationServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_sessions.go", "func newInviteReadServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.InviteReadServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_sessions.go", "func newInviteMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.InviteMutationServiceConfig")
	assertFileContains(t, "campaigns/composition_service_config_creation.go", "func newCharacterCreationServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CharacterCreationServiceConfig")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignAutomationReadService interface")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignAutomationMutationService interface")
	assertFileContains(t, "campaigns/app/service_contracts.go", "CampaignCharacter(context.Context, string, string, CharacterReadContext)")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignCharacterReadService interface")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignCharacterControlService interface")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignCharacterMutationService interface")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "type CampaignGameService interface")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "type Service interface")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "type ServiceConfig struct")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "type AutomationReadServiceConfig struct")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "type AutomationMutationServiceConfig struct")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "func NewService(config ServiceConfig)")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "type catalogService struct")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "func NewAutomationReadService(")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "func NewAutomationMutationService(")
	assertFileMissing(t, "campaigns/app/service_config.go")
	assertFileContains(t, "campaigns/service_configs.go", "type serviceConfigs struct")
	assertFileContains(t, "campaigns/service_configs.go", "type pageServiceConfig struct")
	assertFileContains(t, "campaigns/service_configs.go", "type overviewServiceConfig struct")
	assertFileContains(t, "campaigns/service_configs.go", "type characterServiceConfig struct")
	assertFileContains(t, "campaigns/service_configs.go", "type inviteServiceConfig struct")
	assertFileMissing(t, "campaigns/app/service_builders.go")
	assertFileContains(t, "campaigns/app/service_config_people.go", "type AutomationReadServiceConfig struct")
	assertFileContains(t, "campaigns/app/service_config_people.go", "type AutomationMutationServiceConfig struct")
	assertFileContains(t, "campaigns/app/service_config_people.go", "type CharacterReadServiceConfig struct")
	assertFileContains(t, "campaigns/app/service_config_people.go", "type CharacterControlServiceConfig struct")
	assertFileContains(t, "campaigns/app/service_config_people.go", "type CharacterMutationServiceConfig struct")
	assertFileContains(t, "campaigns/app/service_config_people.go", "Mutation     CampaignCharacterControlMutationGateway")
	assertFileContains(t, "campaigns/app/service_config_people.go", "Mutation CampaignCharacterMutationGateway")
	assertFileContains(t, "campaigns/app/service_builders_core.go", "type catalogService struct")
	assertFileContains(t, "campaigns/app/service_builders_core.go", "type authorizationSupport struct")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "func NewAutomationReadService(config AutomationReadServiceConfig, authorization AuthorizationGateway) CampaignAutomationReadService")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "func NewAutomationMutationService(config AutomationMutationServiceConfig, authorization AuthorizationGateway) CampaignAutomationMutationService")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "func NewCharacterReadService(config CharacterReadServiceConfig, authorization AuthorizationGateway)")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "func NewCharacterControlService(config CharacterControlServiceConfig, authorization AuthorizationGateway)")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "func NewCharacterMutationService(config CharacterMutationServiceConfig, authorization AuthorizationGateway)")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "mutation     CampaignCharacterControlMutationGateway")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "mutation CampaignCharacterMutationGateway")
	assertFileContains(t, "campaigns/app/service_builders_creation.go", "func NewCharacterCreationPageService(config CharacterCreationServiceConfig)")
	assertFileContains(t, "campaigns/app/service_builders_creation.go", "func NewCharacterCreationMutationService(config CharacterCreationServiceConfig, authorization AuthorizationGateway)")
	assertFileMissing(t, "campaigns/app/service_exports.go")
	assertFileContains(t, "campaigns/app/service_reads.go", "func (s catalogService) ListCampaigns(")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignParticipantReadService interface {")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignParticipantMutationService interface {")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignSessionReadService interface {")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignSessionMutationService interface {")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignInviteReadService interface {")
	assertFileContains(t, "campaigns/app/service_contracts.go", "type CampaignInviteMutationService interface {")
	assertFileContains(t, "campaigns/app/service_config_people.go", "type ParticipantReadServiceConfig struct {")
	assertFileContains(t, "campaigns/app/service_config_people.go", "type ParticipantMutationServiceConfig struct {")
	assertFileContains(t, "campaigns/app/service_config_sessions.go", "type SessionReadServiceConfig struct {")
	assertFileContains(t, "campaigns/app/service_config_sessions.go", "type SessionMutationServiceConfig struct {")
	assertFileContains(t, "campaigns/app/service_config_sessions.go", "type InviteReadServiceConfig struct {")
	assertFileContains(t, "campaigns/app/service_config_sessions.go", "type InviteMutationServiceConfig struct {")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "func NewParticipantReadService(config ParticipantReadServiceConfig, authorization AuthorizationGateway) CampaignParticipantReadService")
	assertFileContains(t, "campaigns/app/service_builders_people.go", "func NewParticipantMutationService(config ParticipantMutationServiceConfig, authorization AuthorizationGateway) CampaignParticipantMutationService")
	assertFileContains(t, "campaigns/app/service_builders_sessions.go", "func NewSessionReadService(config SessionReadServiceConfig) CampaignSessionReadService")
	assertFileContains(t, "campaigns/app/service_builders_sessions.go", "func NewSessionMutationService(config SessionMutationServiceConfig, authorization AuthorizationGateway) CampaignSessionMutationService")
	assertFileContains(t, "campaigns/app/service_builders_sessions.go", "func NewInviteReadService(config InviteReadServiceConfig, authorization AuthorizationGateway) CampaignInviteReadService")
	assertFileContains(t, "campaigns/app/service_builders_sessions.go", "func NewInviteMutationService(config InviteMutationServiceConfig, authorization AuthorizationGateway) CampaignInviteMutationService")
	assertFileContains(t, "campaigns/app/service_entities_participants.go", "func (s participantReadService) CampaignParticipantEditor(")
	assertFileContains(t, "campaigns/app/service_mutations_participants.go", "func (s participantMutationService) CreateParticipant(")
	assertFileContains(t, "campaigns/app/service_entities_characters.go", "func (s characterReadService) CampaignCharacterEditor(")
	assertFileContains(t, "campaigns/app/service_character_control.go", "func (s characterControlService) CampaignCharacterControl(")
	assertFileContains(t, "campaigns/app/service_mutations_character_control.go", "func (s characterControlService) SetCharacterController(")
	assertFileContains(t, "campaigns/app/service_mutations_characters.go", "func (s characterMutationService) CreateCharacter(")
	assertFileContains(t, "campaigns/app/service_entities_sessions.go", "func (s sessionReadService) CampaignSessions(")
	assertFileContains(t, "campaigns/app/service_mutations_sessions.go", "func (s sessionMutationService) StartSession(")
	assertFileContains(t, "campaigns/app/service_entities_invites.go", "func (s inviteReadService) CampaignInvites(")
	assertFileContains(t, "campaigns/app/service_mutations_invites.go", "func (s inviteMutationService) CreateInvite(")
	assertFileContains(t, "campaigns/app/service_creation.go", "func (s creationPageService) CampaignCharacterCreationCatalog(")
	assertFileContains(t, "campaigns/app/service_test_helpers_test.go", "type testGatewayBundle interface")
	assertFileContains(t, "campaigns/app/service_test_helpers_test.go", "type testServiceBundle struct")
	assertFileContains(t, "campaigns/app/service_test_helpers_test.go", "func newService(gateway testGatewayBundle) testServiceBundle")
	assertFileMissing(t, "campaigns/app/service_reads_game_test.go")
	assertFileMissing(t, "campaigns/app/types_game_surface.go")
	assertFileDoesNotContain(t, "campaigns/app/gateway_contracts.go", "type CampaignGateway interface")
	assertFileContains(t, "campaigns/app/gateway_contracts.go", "type CampaignAutomationReadGateway interface")
	assertFileContains(t, "campaigns/app/gateway_contracts.go", "type CampaignAutomationMutationGateway interface")
	assertFileContains(t, "campaigns/app/gateway_contracts.go", "type CampaignCharacterControlMutationGateway interface")
	assertFileDoesNotContain(t, "campaigns/app/gateway_contracts.go", "type CampaignGameReadGateway interface")
	assertFileContains(t, "campaigns/app/gateway_contracts.go", "CampaignCharacter(context.Context, string, string, CharacterReadContext)")
	assertFileContains(t, "campaigns/app/gateway_contracts.go", "SetCharacterController(context.Context, string, string, string) error")
	assertFileContains(t, "campaigns/app/gateway_contracts.go", "CreateCharacter(context.Context, string, CreateCharacterInput)")
	assertFileContains(t, "campaigns/app/service_ai_binding.go", "func (s automationReadService) campaignAIBindingSummary(")
	assertFileContains(t, "campaigns/app/service_ai_binding.go", "func (s automationReadService) campaignAIBindingSettings(")
	assertFileContains(t, "campaigns/app/service_ai_binding.go", "func (s automationMutationService) updateCampaignAIBinding(")
	assertFileDoesNotContain(t, "campaigns/handlers_chat.go", "h.workspace.CampaignGameSurface(")
	assertFileDoesNotContain(t, "campaigns/handlers_chat.go", "h.game.CampaignGameSurface(")
	assertFileContains(t, "campaigns/handlers_chat.go", "playlaunchgrant.Issue(")
	assertFileContains(t, "campaigns/handlers_chat.go", "playorigin.PlayURL(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_participants.go", "h.automationReads.CampaignAIBindingEditor(")
	assertFileContains(t, "campaigns/handlers_detail_overview.go", "h.overview.automationReads.CampaignAIBindingSummary(ctx, campaignID, page.workspace.AIAgentID, page.workspace.GMMode)")
	assertFileContains(t, "campaigns/handlers_detail_overview.go", "h.overview.automationReads.CampaignAIBindingSettings(ctx, campaignID, page.workspace.AIAgentID)")
	assertFileContains(t, "campaigns/handlers_mutation.go", "h.overview.automationMutate.UpdateCampaignAIBinding(ctx, campaignID, input)")
	assertFileDoesNotContain(t, "campaigns/campaign_page_context.go", "h.workspace.CampaignSessions(")
	assertFileContains(t, "campaigns/campaign_page_context.go", "h.pages.sessionReads.CampaignSessions(")
	assertFileContains(t, "campaigns/handlers_detail_sessions_invites.go", "h.pages.sessionReads.CampaignSessionReadiness(")
	assertFileContains(t, "campaigns/test_config_test.go", "func serviceConfigsWithGRPCDeps(")
	assertFileContains(t, "campaigns/test_config_test.go", "return newServiceConfigsFromGRPCDeps(deps, assetBaseURL)")
	assertFileDoesNotContain(t, "campaigns/test_config_test.go", "campaigngateway.NewGRPCGateway(")
	assertFileDoesNotContain(t, "campaigns/module_test.go", "campaigngateway.NewGRPCGateway(")
	assertFileDoesNotContain(t, "campaigns/handlers_list_create_test.go", "campaigngateway.NewGRPCGateway(")
	assertFileDoesNotContain(t, "campaigns/handlers_detail_test.go", "campaigngateway.NewGRPCGateway(")
	assertFileMissing(t, "campaigns/gateway/grpc.go")
	assertFileMissing(t, "campaigns/gateway/grpc_interaction.go")
	assertFileContains(t, "campaigns/gateway/grpc_contracts.go", "type CampaignReadClient interface")
	assertFileDoesNotContain(t, "campaigns/gateway/grpc_contracts.go", "type InteractionClient interface")
	assertFileDoesNotContain(t, "campaigns/gateway/grpc_config.go", "func NewGRPCGateway(deps GRPCGatewayDeps)")
	assertFileDoesNotContain(t, "campaigns/gateway/grpc_config.go", "type GRPCGatewayReadDeps struct")
	assertFileDoesNotContain(t, "campaigns/gateway/grpc_config.go", "type GRPCGatewayMutationDeps struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "type CatalogReadDeps struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "type CharacterReadDeps struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "type CharacterControlMutationDeps struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "type InviteReadDeps struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "type GRPCGatewayDeps struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "type automationReadGateway struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "CharacterControl  CharacterControlMutationDeps")
	assertFileDoesNotContain(t, "campaigns/gateway/grpc_config.go", "type GameReadDeps struct")
	assertFileDoesNotContain(t, "campaigns/gateway/grpc_config.go", "func NewGameReadGateway(")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "type characterControlMutationGateway struct")
	assertFileContains(t, "campaigns/gateway/grpc_config.go", "func NewCharacterControlMutationGateway(mutationDeps CharacterControlMutationDeps)")
	assertFileContains(t, "campaigns/gateway/grpc_character_control.go", "func (g characterControlMutationGateway) SetCharacterController(")
	assertFileContains(t, "campaigns/composition_gateway_deps.go", "CatalogRead: campaigngateway.CatalogReadDeps{")
	assertFileContains(t, "campaigns/composition_gateway_deps.go", "CharacterControl: campaigngateway.CharacterControlMutationDeps{")
	assertFileContains(t, "campaigns/composition_gateway_deps.go", "InviteRead: campaigngateway.InviteReadDeps{")
	assertFileDoesNotContain(t, "campaigns/composition_gateway_deps.go", "Read: campaigngateway.GRPCGatewayReadDeps{")
	assertFileDoesNotContain(t, "campaigns/composition_gateway_deps.go", "Mutation: campaigngateway.GRPCGatewayMutationDeps{")
	assertFileContains(t, "campaigns/gateway/grpc_test_helpers_test.go", "type testGatewayBundle interface")
	assertFileContains(t, "campaigns/gateway/grpc_test_helpers_test.go", "type GRPCGatewayReadDeps struct")
	assertFileContains(t, "campaigns/gateway/grpc_test_helpers_test.go", "func NewGRPCGateway(deps GRPCGatewayDeps) testGatewayBundle")
	assertFileContains(t, "campaigns/render/detail_characters.go", "Character       CharacterView")
	assertFileContains(t, "campaigns/render/detail_characters.templ", "view.Character.")
	assertFileDoesNotContain(t, "campaigns/render/detail_characters.templ", "campaignCharacterByID(view.CharacterID, view.Characters)")
}

func TestCampaignCharacterCreationWorkflowOwnershipStaysOutOfApp(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "campaigns/workflow/service.go", "type PageAppService interface")
	assertFileContains(t, "campaigns/workflow/service.go", "type MutationAppService interface")
	assertFileContains(t, "campaigns/workflow/service.go", "func (s PageService) LoadPage(")
	assertFileContains(t, "campaigns/workflow/service.go", "func (s MutationService) ApplyStep(")
	assertFileContains(t, "campaigns/workflow/service.go", "func NewPageService(app PageAppService, workflows Registry) PageService")
	assertFileContains(t, "campaigns/workflow/service.go", "func NewMutationService(app MutationAppService, workflows Registry) MutationService")
	assertFileContains(t, "campaigns/workflow/service.go", "workflow.BuildView(progress, catalog, profile)")
	assertFileContains(t, "campaigns/workflow/types.go", "type Progress struct")
	assertFileContains(t, "campaigns/workflow/types.go", "type Catalog struct")
	assertFileContains(t, "campaigns/workflow/types.go", "type Profile struct")
	assertFileContains(t, "campaigns/workflow/view_types.go", "type CharacterCreationView struct")
	assertFileContains(t, "campaigns/workflow/app_adapter.go", "func NewPageAppService(app campaignapp.CampaignCharacterCreationPageService) PageAppService")
	assertFileContains(t, "campaigns/workflow/app_adapter.go", "func NewMutationAppService(app campaignapp.CampaignCharacterCreationMutationService) MutationAppService")
	assertFileContains(t, "campaigns/render/creation_adapter.go", "func NewCharacterCreationView(")
	assertFileContains(t, "campaigns/render/creation_adapter.go", "func NewCharacterCreationPageView(")
	assertFileContains(t, "campaigns/handlers.go", "creationPages := campaignworkflow.NewPageAppService(")
	assertFileContains(t, "campaigns/handlers.go", "creationFlow := campaignworkflow.NewMutationAppService(")
	assertFileContains(t, "campaigns/handlers.go", "Pages: creationPages")
	assertFileContains(t, "campaigns/handlers.go", "Flow:  creationFlow")
	assertFileDoesNotContain(t, "campaigns/workflow/service.go", "type AppService interface")
	assertFileDoesNotContain(t, "campaigns/workflow/service.go", "type Service struct")
	assertFileDoesNotContain(t, "campaigns/workflow/service.go", "func NewService(app AppService, workflows Registry) Service")
	assertFileDoesNotContain(t, "campaigns/workflow/contract.go", "campaignrender.")
	assertFileDoesNotContain(t, "campaigns/workflow/service.go", "campaignrender.")
	assertFileDoesNotContain(t, "campaigns/workflow/contract.go", "campaignapp.CampaignCharacterCreationProgress")
	assertFileDoesNotContain(t, "campaigns/workflow/contract.go", "campaignapp.CampaignCharacterCreationCatalog")
	assertFileDoesNotContain(t, "campaigns/workflow/contract.go", "campaignapp.CampaignCharacterCreationProfile")
	assertFileDoesNotContain(t, "campaigns/app/service_contracts.go", "CampaignCharacterCreation(context.Context, string, string, language.Tag")
	assertFileDoesNotContain(t, "campaigns/app/types_character_creation.go", "type CampaignCharacterCreation struct")
	assertFileDoesNotContain(t, "campaigns/handlers_workflow.go", "workflow.ParseStepInput(")
	assertFileDoesNotContain(t, "campaigns/handlers_creation_page.go", "CampaignCharacterCreation(")
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
	assertFileMissing(t, "discovery/gateway.go")
	assertFileDoesNotContain(t, "discovery/gateway/grpc.go", "type StarterEntry =")
	assertFileDoesNotContain(t, "discovery/gateway/grpc.go", "type Gateway =")
}

func TestMalformedJSONHandlingUsesSharedPlatformSeam(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "../platform/jsoninput/jsoninput.go", "func DecodeStrictInvalidInput(")
	assertFileContains(t, "publicauth/handlers_inputs.go", "jsoninput.DecodeStrictInvalidInput(")
	assertFileContains(t, "settings/handlers_inputs.go", "jsoninput.DecodeStrictInvalidInput(")
	assertFileContains(t, "campaigns/handlers_invite_search.go", "jsoninput.DecodeStrictInvalidInput(")
	assertFileDoesNotContain(t, "publicauth/handlers_inputs.go", "Invalid JSON body.")
	assertFileDoesNotContain(t, "settings/handlers_inputs.go", "Invalid JSON body.")
	assertFileDoesNotContain(t, "campaigns/handlers_invite_search.go", "Invalid JSON body.")
}

func TestFormParsingUsesSharedPlatformSeams(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "../platform/forminput/forminput.go", "func ParseInvalidInput(")
	assertFileContains(t, "../platform/forminput/forminput.go", "func ParseOrRedirectErrorNotice(")
	assertFileContains(t, "settings/handlers_profile.go", "forminput.ParseInvalidInput(")
	assertFileContains(t, "settings/handlers_locale.go", "forminput.ParseInvalidInput(")
	assertFileContains(t, "settings/handlers_ai_keys.go", "forminput.ParseInvalidInput(")
	assertFileContains(t, "settings/handlers_ai_agents.go", "forminput.ParseInvalidInput(")
	assertFileContains(t, "campaigns/handlers_list_create.go", "forminput.ParseOrRedirectErrorNotice(")
	assertFileContains(t, "campaigns/handlers_mutation.go", "forminput.ParseOrRedirectErrorNotice(")
	assertFileContains(t, "campaigns/handlers_workflow.go", "forminput.ParseOrRedirectErrorNotice(")
	assertFileDoesNotContain(t, "settings/handlers_profile.go", "r.ParseForm()")
	assertFileDoesNotContain(t, "settings/handlers_locale.go", "r.ParseForm()")
	assertFileDoesNotContain(t, "settings/handlers_ai_keys.go", "r.ParseForm()")
	assertFileDoesNotContain(t, "settings/handlers_ai_agents.go", "r.ParseForm()")
	assertFileDoesNotContain(t, "campaigns/handlers_list_create.go", "r.ParseForm()")
	assertFileDoesNotContain(t, "campaigns/handlers_mutation.go", "requireParsedForm(")
	assertFileDoesNotContain(t, "campaigns/handlers_mutation.go", "r.ParseForm()")
	assertFileDoesNotContain(t, "campaigns/handlers_workflow.go", "requireParsedForm(")
	assertFileDoesNotContain(t, "campaigns/handlers_workflow.go", "r.ParseForm()")
	assertFileDoesNotContain(t, "campaigns/campaign_page_context.go", "parseFormOrWriteError(")
	assertFileMissing(t, "campaigns/handlers_form.go")
}

func TestRequestResolutionUsesSharedPlatformSeam(t *testing.T) {
	t.Parallel()

	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "type PageResolver interface")
	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "type PrincipalResolver interface")
	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "type Principal struct")
	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "type Base struct")
	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "type LocalizedPage struct")
	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "func NewFromPageResolver(")
	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "func ResolveLocalizedPage(")
	assertFileContains(t, "../platform/requestresolver/requestresolver.go", "func ResolveViewer(")
	assertFileContains(t, "../platform/modulehandler/modulehandler.go", "requestresolver.Base")
	assertFileContains(t, "../platform/modulehandler/modulehandler.go", "func NewBaseFromPrincipal(")
	assertFileContains(t, "../platform/modulehandler/modulehandler.go", "requestresolver.NewFromPageResolver(")
	assertFileContains(t, "../platform/modulehandler/modulehandler.go", "requestresolver.ResolveLocalizedPage(")
	assertFileContains(t, "../platform/publichandler/publichandler.go", "requestresolver.Base")
	assertFileContains(t, "../platform/publichandler/publichandler.go", "requestresolver.NewFromPageResolver(")
	assertFileContains(t, "../platform/publichandler/publichandler.go", "func (b Base) PageLocalizer(")
	assertFileContains(t, "../platform/publichandler/publichandler.go", "func (b Base) RequestUserID(")
	assertFileContains(t, "../platform/pagerender/pagerender.go", "resolver requestresolver.PageResolver")
	assertFileContains(t, "../platform/pagerender/pagerender.go", "requestresolver.ResolveLocalizedPage(")
	assertFileContains(t, "../platform/pagerender/pagerender.go", "requestresolver.ResolveViewer(")
	assertFileContains(t, "../platform/weberror/weberror.go", "resolver requestresolver.PageResolver")
	assertFileContains(t, "../platform/weberror/weberror.go", "requestresolver.ResolveLocalizedPage(")
	assertFileContains(t, "../platform/weberror/weberror.go", "requestresolver.ResolveViewer(")
	assertFileContains(t, "discovery/handlers.go", "h.PageLocalizer(")
	assertFileContains(t, "invite/handlers.go", "h.PageLocalizer(")
	assertFileContains(t, "invite/handlers.go", "h.RequestUserID(")
	assertFileContains(t, "profile/handlers.go", "h.PageLocalizer(")
	assertFileContains(t, "publicauth/handlers_passkey.go", "h.PageLocalizer(")
	assertFileDoesNotContain(t, "discovery/handlers.go", "requestresolver.ResolveLocalizedPage(")
	assertFileDoesNotContain(t, "invite/handlers.go", "requestresolver.ResolveLocalizedPage(")
	assertFileDoesNotContain(t, "profile/handlers.go", "requestresolver.ResolveLocalizedPage(")
	assertFileDoesNotContain(t, "publicauth/handlers_passkey.go", "requestresolver.ResolveLocalizedPage(")
	assertFileContains(t, "../composition/compose.go", "Principal requestresolver.PrincipalResolver")
	assertFileDoesNotContain(t, "../server.go", "requestresolver.NewPrincipal(")
	assertFileContains(t, "registry.go", "Principal        requestresolver.PrincipalResolver")
	assertFileDoesNotContain(t, "../platform/pagerender/pagerender.go", "type RequestResolver interface")
	assertFileDoesNotContain(t, "../composition/compose.go", "type PrincipalResolvers struct")
	assertFileDoesNotContain(t, "module.go", "type ModuleResolvers struct")
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

func assertFileMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("%s exists; expected it to be missing", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v", path, err)
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
