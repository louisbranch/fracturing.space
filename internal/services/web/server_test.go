package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/icons"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	"google.golang.org/grpc"
)

func TestNewServerRequiresHTTPAddr(t *testing.T) {
	t.Parallel()

	_, err := NewServer(context.Background(), Config{})
	if err == nil {
		t.Fatalf("expected error for empty HTTPAddr")
	}
}

func TestNewHandlerMountsOnlyStableModulesByDefault(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	publicRR := httptest.NewRecorder()
	h.ServeHTTP(publicRR, publicReq)
	if publicRR.Code != http.StatusOK {
		t.Fatalf("public status = %d, want %d", publicRR.Code, http.StatusOK)
	}

	publicProfileReq := httptest.NewRequest(http.MethodGet, "/u/alice", nil)
	publicProfileRR := httptest.NewRecorder()
	h.ServeHTTP(publicProfileRR, publicProfileReq)
	if publicProfileRR.Code != http.StatusServiceUnavailable {
		t.Fatalf("public profile status = %d, want %d", publicProfileRR.Code, http.StatusServiceUnavailable)
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	protectedRR := httptest.NewRecorder()
	h.ServeHTTP(protectedRR, protectedReq)
	if protectedRR.Code != http.StatusFound {
		t.Fatalf("protected status = %d, want %d", protectedRR.Code, http.StatusFound)
	}

	dashboardReq := httptest.NewRequest(http.MethodGet, "/app/dashboard/", nil)
	dashboardRR := httptest.NewRecorder()
	h.ServeHTTP(dashboardRR, dashboardReq)
	if dashboardRR.Code != http.StatusFound {
		t.Fatalf("dashboard status = %d, want %d", dashboardRR.Code, http.StatusFound)
	}

	dashboardNoSlashReq := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	dashboardNoSlashRR := httptest.NewRecorder()
	h.ServeHTTP(dashboardNoSlashRR, dashboardNoSlashReq)
	if dashboardNoSlashRR.Code != http.StatusFound {
		t.Fatalf("dashboard (no slash) status = %d, want %d", dashboardNoSlashRR.Code, http.StatusFound)
	}

	campaignsReq := httptest.NewRequest(http.MethodGet, "/app/campaigns/123", nil)
	campaignsRR := httptest.NewRecorder()
	h.ServeHTTP(campaignsRR, campaignsReq)
	if campaignsRR.Code != http.StatusFound {
		t.Fatalf("campaigns status = %d, want %d", campaignsRR.Code, http.StatusFound)
	}
	if got := campaignsRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("campaigns redirect = %q, want %q", got, "/login")
	}
	if got := dashboardRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("dashboard redirect = %q, want %q", got, "/login")
	}
	if got := dashboardNoSlashRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("dashboard (no slash) redirect = %q, want %q", got, "/login")
	}

	experimentalProtectedReq := httptest.NewRequest(http.MethodGet, "/app/notifications/", nil)
	experimentalProtectedRR := httptest.NewRecorder()
	h.ServeHTTP(experimentalProtectedRR, experimentalProtectedReq)
	if experimentalProtectedRR.Code != http.StatusNotFound {
		t.Fatalf("experimental protected status = %d, want %d", experimentalProtectedRR.Code, http.StatusNotFound)
	}
}

func TestNewHandlerMountsExperimentalModulesWhenEnabled(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{EnableExperimentalModules: true})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	publicRR := httptest.NewRecorder()
	h.ServeHTTP(publicRR, publicReq)
	if publicRR.Code != http.StatusOK {
		t.Fatalf("public status = %d, want %d", publicRR.Code, http.StatusOK)
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/app/notifications/", nil)
	protectedRR := httptest.NewRecorder()
	h.ServeHTTP(protectedRR, protectedReq)
	if protectedRR.Code != http.StatusFound {
		t.Fatalf("protected status = %d, want %d", protectedRR.Code, http.StatusFound)
	}

	campaignsReq := httptest.NewRequest(http.MethodGet, "/app/campaigns/123", nil)
	campaignsRR := httptest.NewRecorder()
	h.ServeHTTP(campaignsRR, campaignsReq)
	if campaignsRR.Code != http.StatusFound {
		t.Fatalf("campaigns status = %d, want %d", campaignsRR.Code, http.StatusFound)
	}
}

func TestDefaultCampaignStableSurfaceHidesScaffoldDetailRoutes(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultStableProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	for _, path := range []string{
		"/app/campaigns/c1/sessions",
		"/app/campaigns/c1/sessions/sess-1",
		"/app/campaigns/c1/characters/char-1",
		"/app/campaigns/c1/invites",
		"/app/campaigns/c1/game",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		attachSessionCookie(t, req, auth, "user-1")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusNotFound)
		}
	}
}

func TestExperimentalCampaignSurfaceExposesDetailRoutes(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	for _, path := range []string{
		"/app/campaigns/c1/sessions",
		"/app/campaigns/c1/sessions/sess-1",
		"/app/campaigns/c1/characters/char-1",
		"/app/campaigns/c1/invites",
		"/app/campaigns/c1/game",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		attachSessionCookie(t, req, auth, "user-1")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q status = %d, want %d", path, rr.Code, http.StatusOK)
		}
	}
}

func TestExperimentalCampaignMutationRouteRejectsMemberAccess(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{
		EnableExperimentalModules: true,
		AuthClient:                auth,
		CampaignClient:            defaultCampaignClient(),
		ParticipantClient: fakeWebParticipantClient{response: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{{
			Id:             "p-member",
			CampaignId:     "c1",
			UserId:         "user-1",
			CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
		}}}},
		SocialClient:     defaultSocialClient(),
		CredentialClient: fakeCredentialClient{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/c1/sessions/start", nil)
	req.Header.Set("Origin", "http://example.com")
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestProtectedRouteDoesNotTrustUserHeader(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.Header.Set("X-Web-User", "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}

func TestNewHandlerAddsRequestIDHeader(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Request-ID"); got == "" {
		t.Fatalf("expected response request id header")
	}
}

func TestNewHandlerUsesConfiguredCampaignClient(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{EnableExperimentalModules: true, AuthClient: auth, CampaignClient: fakeCampaignClient{response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Remote"}}}}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Remote") {
		t.Fatalf("body = %q, want configured campaign response", body)
	}
}

func TestAppCampaignsPageRendersPrimaryNavigation(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	assertPrimaryNavLinks(t, rr.Body.String())
}

func TestAppDashboardPageRendersPrimaryNavigation(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/dashboard/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{"dashboard-root"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing dashboard marker %q", marker)
		}
	}
	assertPrimaryNavLinks(t, body)
}

func TestAppSettingsPageRendersPrimaryNavigation(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	assertPrimaryNavLinks(t, rr.Body.String())
}

func TestPrivateSettingsUsesAuthenticatedUserLocaleForShellAndContent(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}}
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{AuthClient: auth, AccountClient: account, CampaignClient: defaultCampaignClient(), SocialClient: defaultSocialClient(), CredentialClient: fakeCredentialClient{}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !account.getProfileCalled {
		t.Fatalf("expected account profile lookup for authenticated locale")
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<html lang="pt-BR"`,
		`>Configurações</a>`,
		`>Campanhas</a>`,
		`<h1 class="mb-0">Configurações</h1>`,
		`<h2 class="card-title">Perfil público</h2>`,
		`<span class="label-text">Nome de usuário</span>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing pt-BR marker %q: %q", marker, body)
		}
	}
}

func TestPrivateSettingsValidationErrorUsesAuthenticatedUserLocale(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}}
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{AuthClient: auth, AccountClient: account, CampaignClient: defaultCampaignClient(), SocialClient: defaultSocialClient(), CredentialClient: fakeCredentialClient{}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	form := url.Values{"name": {"Rhea"}}
	form.Set("name", strings.Repeat("x", 65))
	req := httptest.NewRequest(http.MethodPost, "/app/settings/profile", strings.NewReader(form.Encode()))
	attachSessionCookie(t, req, auth, "user-1")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<html lang="pt-BR"`,
		"Nome deve ter no máximo 64 caracteres.",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing pt-BR validation marker %q: %q", marker, body)
		}
	}
}

func TestAppSettingsRootRedirectsToProfile(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/app/settings/profile" {
		t.Fatalf("Location = %q, want %q", got, "/app/settings/profile")
	}
}

func TestAppSettingsProfileRendersSettingsMenuAndContent(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<h1 class="mb-0">Settings</h1>`,
		`id="settings-profile"`,
		`href="/app/settings/profile"`,
		`href="/app/settings/locale"`,
		`href="/app/settings/ai-keys"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing settings marker %q: %q", marker, body)
		}
	}
}

func TestPrimaryNavigationOmitsExperimentalLinksByDefault(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, `href="/app/notifications"`) {
		t.Fatalf("body unexpectedly includes experimental notifications link: %q", body)
	}
	if strings.Contains(body, `href="/app/profile"`) {
		t.Fatalf("body unexpectedly includes experimental profile link: %q", body)
	}
}

func TestPrimaryNavigationUsesDashboardAndCampaignIcons(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="/app/dashboard"`) {
		t.Fatalf("body missing dashboard nav link")
	}
	dashboardIconHref := `href="#` + icons.LucideSymbolID("layout-dashboard") + `"`
	if !strings.Contains(body, dashboardIconHref) {
		t.Fatalf("body missing dashboard icon %q", dashboardIconHref)
	}
	dashboardIconSymbol := `id="` + icons.LucideSymbolID("layout-dashboard") + `"`
	if !strings.Contains(body, dashboardIconSymbol) {
		t.Fatalf("body missing dashboard icon symbol %q", dashboardIconSymbol)
	}
	if !strings.Contains(body, `href="/app/campaigns"`) {
		t.Fatalf("body missing campaigns nav link")
	}
	campaignIconHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_CAMPAIGN)) + `"`
	if !strings.Contains(body, campaignIconHref) {
		t.Fatalf("body missing campaigns icon %q", campaignIconHref)
	}
}

func TestAppPageTitleUsesWebComposition(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `<title>Public Profile | Fracturing.Space</title>`) {
		t.Fatalf("body missing composed page title: %q", body)
	}
}

func TestPrimaryNavigationOmitsInvitesLinkWhileScaffoldDisabled(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	// Invariant: invites scaffolding is intentionally hidden until a real invites feature is enabled.
	if strings.Contains(rr.Body.String(), `href="/app/invites"`) {
		t.Fatalf("body unexpectedly includes invites nav link: %q", rr.Body.String())
	}
}

func TestInvitesRouteReturnsNotFoundWhileScaffoldDisabled(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/invites", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestUnknownRootRouteRendersNotFoundPage(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{AuthClient: newFakeWebAuthClient()})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/123", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("content-type = %q, want text/html", got)
	}
	body := rr.Body.String()
	for _, marker := range []string{`id="auth-shell"`, `id="app-error-state"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing route-miss marker %q: %q", marker, body)
		}
	}
	// Invariant: unknown routes should use the shared HTML not-found surface, never net/http plain text.
	if strings.Contains(body, "404 page not found") {
		t.Fatalf("body unexpectedly rendered plain 404 text: %q", body)
	}
}

func TestLoginPageIncludesAuthShellAndPasskeyEndpoints(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{AuthClient: newFakeWebAuthClient()})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`id="auth-shell"`,
		`id="auth-language-menu"`,
		`/passkeys/login/start`,
		`/passkeys/login/finish`,
		`/passkeys/register/start`,
		`/passkeys/register/finish`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing auth contract marker %q", marker)
		}
	}
}

func TestLoginPageLocaleMenuUsesConsistentLabels(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{AuthClient: newFakeWebAuthClient()})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	tests := []struct {
		path        string
		wantTrigger string
	}{
		{path: "/login?lang=en-US", wantTrigger: "EN"},
		{path: "/login?lang=pt-BR", wantTrigger: "PT-BR"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			if !strings.Contains(body, `id="auth-language-menu"`) {
				t.Fatalf("body missing auth language menu: %q", body)
			}
			if !strings.Contains(body, `backdrop-blur">`+tc.wantTrigger+`</div>`) {
				t.Fatalf("body missing locale trigger label %q: %q", tc.wantTrigger, body)
			}
			for _, marker := range []string{`data-lang="en-US"`, `>EN</a>`, `data-lang="pt-BR"`, `>PT-BR</a>`} {
				if !strings.Contains(body, marker) {
					t.Fatalf("body missing locale option marker %q: %q", marker, body)
				}
			}
		})
	}
}

func TestAppPageIncludesThemeAssets(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{`/static/theme.css`, `/static/app-shell.js`, `data-layout="app"`, `id="main"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing app shell marker %q", marker)
		}
	}
}

func TestAppPageUsesWebStyleChromeMarkers(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{`Fracturing.Space`, `data-layout="app"`, `id="main"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing app chrome contract marker %q", marker)
		}
	}
	assertPrimaryNavLinks(t, body)
}

func TestAppLayoutIncludesHTMXErrorSwapContract(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(defaultProtectedConfig(auth))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{`src="/static/app-shell.js"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing app shell script marker %q", marker)
		}
	}
}

func TestAppPageRendersUserDropdownFromSocial(t *testing.T) {
	t.Parallel()

	social := &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Username: "rhea", Name: "Rhea Vale", AvatarSetId: "avatar_set_v1", AvatarAssetId: "001"}}}
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{AuthClient: auth, SocialClient: social, AssetBaseURL: "https://cdn.example.com/avatars", AccountClient: &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}}, CampaignClient: defaultCampaignClient()})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !social.getUserProfileCalled {
		t.Fatalf("expected social profile lookup")
	}
	for _, marker := range []string{
		`src="https://cdn.example.com/avatars/001.png"`,
		`alt="Rhea Vale"`,
		`href="/u/rhea"`,
		`href="/app/settings"`,
		`action="/logout"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing user dropdown contract marker %q: %q", marker, body)
		}
	}
	if strings.Index(body, `href="/u/rhea"`) > strings.Index(body, `href="/app/settings"`) {
		t.Fatalf("expected profile menu item before settings menu item: %q", body)
	}
}

func TestAppPageUserDropdownProfileFallsBackToSettingsNoticeWhenUsernameMissing(t *testing.T) {
	t.Parallel()

	social := &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "Rhea Vale"}}}
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{AuthClient: auth, SocialClient: social, AssetBaseURL: "https://cdn.example.com/avatars", AccountClient: &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}}, CampaignClient: defaultCampaignClient()})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `href="/app/settings/profile?notice=public-profile-required"`) {
		t.Fatalf("body missing profile fallback notice link: %q", body)
	}
}

func TestAppPageUsesDeterministicAvatarWhenProfileHasNoAssetSelection(t *testing.T) {
	t.Parallel()

	social := &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "Rhea Vale"}}}
	assetBaseURL := "https://cdn.example.com/avatars"
	expectedAvatarURL := websupport.AvatarImageURL(assetBaseURL, "user", "user-1", "", "")
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{AuthClient: auth, SocialClient: social, AssetBaseURL: assetBaseURL, AccountClient: &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}}, CampaignClient: defaultCampaignClient()})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`src="` + expectedAvatarURL + `"`,
		`alt="Rhea Vale"`,
		`class="rounded-full"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing deterministic avatar marker %q: %q", marker, body)
		}
	}
}

type fakeSocialClient struct {
	getUserProfileResp   *socialv1.GetUserProfileResponse
	getUserProfileErr    error
	getUserProfileCalled bool
}

type fakeAccountClient struct {
	getProfileResp   *authv1.GetProfileResponse
	getProfileErr    error
	getProfileCalled bool
	lastUpdateReq    *authv1.UpdateProfileRequest
	updateErr        error
}

type fakeCredentialClient struct{}

func (f *fakeAccountClient) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	f.getProfileCalled = true
	if f.getProfileErr != nil {
		return nil, f.getProfileErr
	}
	if f.getProfileResp != nil {
		return f.getProfileResp, nil
	}
	return &authv1.GetProfileResponse{}, nil
}

func (f *fakeAccountClient) UpdateProfile(_ context.Context, req *authv1.UpdateProfileRequest, _ ...grpc.CallOption) (*authv1.UpdateProfileResponse, error) {
	f.lastUpdateReq = req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return &authv1.UpdateProfileResponse{}, nil
}

func (f *fakeSocialClient) AddContact(context.Context, *socialv1.AddContactRequest, ...grpc.CallOption) (*socialv1.AddContactResponse, error) {
	return &socialv1.AddContactResponse{}, nil
}

func (f *fakeSocialClient) RemoveContact(context.Context, *socialv1.RemoveContactRequest, ...grpc.CallOption) (*socialv1.RemoveContactResponse, error) {
	return &socialv1.RemoveContactResponse{}, nil
}

func (f *fakeSocialClient) ListContacts(context.Context, *socialv1.ListContactsRequest, ...grpc.CallOption) (*socialv1.ListContactsResponse, error) {
	return &socialv1.ListContactsResponse{}, nil
}

func (f *fakeSocialClient) SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	return &socialv1.SetUserProfileResponse{}, nil
}

func (f *fakeSocialClient) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	f.getUserProfileCalled = true
	if f.getUserProfileErr != nil {
		return nil, f.getUserProfileErr
	}
	if f.getUserProfileResp != nil {
		return f.getUserProfileResp, nil
	}
	return &socialv1.GetUserProfileResponse{}, nil
}

func (f *fakeSocialClient) LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error) {
	return &socialv1.LookupUserProfileResponse{}, nil
}

func (fakeCredentialClient) ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	return &aiv1.ListCredentialsResponse{}, nil
}

func (fakeCredentialClient) CreateCredential(context.Context, *aiv1.CreateCredentialRequest, ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	return &aiv1.CreateCredentialResponse{}, nil
}

func (fakeCredentialClient) RevokeCredential(context.Context, *aiv1.RevokeCredentialRequest, ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	return &aiv1.RevokeCredentialResponse{}, nil
}

func TestNewHandlerResolvesCookieSessionAtMostOncePerRequest(t *testing.T) {
	t.Parallel()

	auth := newCountingWebAuthClient()
	_, _ = auth.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "user-1"})
	h, err := NewHandler(Config{AuthClient: auth, AccountClient: &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}}, SocialClient: defaultSocialClient(), CredentialClient: fakeCredentialClient{}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if auth.GetWebSessionCalls() != 1 {
		t.Fatalf("GetWebSession calls = %d, want %d", auth.GetWebSessionCalls(), 1)
	}
}

func TestNewServerBuildsHTTPServer(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(context.Background(), Config{HTTPAddr: "127.0.0.1:0", AuthClient: newFakeWebAuthClient()})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if srv.httpAddr != "127.0.0.1:0" {
		t.Fatalf("httpAddr = %q, want %q", srv.httpAddr, "127.0.0.1:0")
	}
	if srv.httpServer == nil {
		t.Fatalf("expected http server")
	}
	srv.Close()
}

func TestListenAndServeRejectsNilServer(t *testing.T) {
	t.Parallel()

	var srv *Server
	err := srv.ListenAndServe(context.Background())
	if err == nil {
		t.Fatalf("expected nil server error")
	}
	if !strings.Contains(err.Error(), "web server is nil") {
		t.Fatalf("error = %q, want nil server message", err.Error())
	}
}

func TestListenAndServeRequiresContext(t *testing.T) {
	t.Parallel()

	srv := &Server{httpServer: &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}}
	err := srv.ListenAndServe(nil)
	if err == nil {
		t.Fatalf("expected context-required error")
	}
	if !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("error = %q, want context-required message", err.Error())
	}
}

func TestListenAndServeReturnsServeError(t *testing.T) {
	t.Parallel()

	srv := &Server{httpServer: &http.Server{Addr: "bad address", Handler: http.NotFoundHandler()}}
	err := srv.ListenAndServe(context.Background())
	if err == nil {
		t.Fatalf("expected serve error")
	}
	if !strings.Contains(err.Error(), "serve web http") {
		t.Fatalf("error = %q, want wrapped serve message", err.Error())
	}
}

func TestListenAndServeShutsDownOnContextCancel(t *testing.T) {
	t.Parallel()

	srv := &Server{httpServer: &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ListenAndServe() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for server shutdown")
	}
}

func TestCloseHandlesNilServerAndNilHTTPServer(t *testing.T) {
	t.Parallel()

	var nilServer *Server
	nilServer.Close()

	(&Server{}).Close()
}

func assertPrimaryNavLinks(t *testing.T, body string) {
	t.Helper()
	for _, href := range []string{"/app/dashboard", "/app/campaigns", "/app/settings"} {
		if !strings.Contains(body, "href=\""+href+"\"") {
			t.Fatalf("body missing nav href %q", href)
		}
	}
	if !strings.Contains(body, `action="/logout"`) {
		t.Fatalf("body missing logout form action %q", "/logout")
	}
}

func attachSessionCookie(t *testing.T, req *http.Request, auth *fakeWebAuthClient, userID string) {
	t.Helper()
	if req == nil {
		t.Fatalf("request is required")
	}
	if auth == nil {
		t.Fatalf("auth client is required")
	}
	if strings.TrimSpace(userID) == "" {
		t.Fatalf("user id is required")
	}
	resp, err := auth.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		t.Fatalf("CreateWebSession() error = %v", err)
	}
	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		t.Fatalf("expected non-empty session id")
	}
	req.AddCookie(&http.Cookie{Name: "web_session", Value: sessionID})
}

func defaultProtectedConfig(auth *fakeWebAuthClient) Config {
	return Config{
		EnableExperimentalModules: true,
		AuthClient:                auth,
		CampaignClient:            defaultCampaignClient(),
		ParticipantClient:         defaultParticipantClient(),
		CharacterClient:           defaultCharacterClient(),
		SessionClient:             defaultSessionClient(),
		InviteClient:              defaultInviteClient(),
		AccountClient: &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US},
		}},
		SocialClient:     defaultSocialClient(),
		CredentialClient: fakeCredentialClient{},
	}
}

func defaultStableProtectedConfig(auth *fakeWebAuthClient) Config {
	return Config{
		EnableExperimentalModules: false,
		AuthClient:                auth,
		CampaignClient:            defaultCampaignClient(),
		AccountClient: &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US},
		}},
		SocialClient:     defaultSocialClient(),
		CredentialClient: fakeCredentialClient{},
	}
}

func defaultSocialClient() *fakeSocialClient {
	return &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Username: "adventurer", Name: "Adventurer"}}}
}

func defaultCampaignClient() fakeCampaignClient {
	return fakeCampaignClient{response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Campaign"}}}}
}

func defaultParticipantClient() fakeWebParticipantClient {
	return fakeWebParticipantClient{response: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{{
		Id:             "p1",
		CampaignId:     "c1",
		UserId:         "user-1",
		Name:           "Owner",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
	}}}}
}

func defaultCharacterClient() fakeWebCharacterClient {
	return fakeWebCharacterClient{response: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{
		Id:   "char-1",
		Name: "Aria",
		Kind: statev1.CharacterKind_PC,
	}}}}
}

func defaultSessionClient() fakeWebSessionClient {
	return fakeWebSessionClient{response: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{
		Id:     "sess-1",
		Name:   "Session One",
		Status: statev1.SessionStatus_SESSION_ACTIVE,
	}}}}
}

func defaultInviteClient() fakeWebInviteClient {
	return fakeWebInviteClient{response: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{{
		Id:              "inv-1",
		CampaignId:      "c1",
		ParticipantId:   "p1",
		RecipientUserId: "user-2",
		Status:          statev1.InviteStatus_PENDING,
	}}}}
}

type fakeCampaignClient struct {
	response   *statev1.ListCampaignsResponse
	err        error
	getResp    *statev1.GetCampaignResponse
	getErr     error
	createResp *statev1.CreateCampaignResponse
	createErr  error
}

type fakeWebParticipantClient struct {
	response *statev1.ListParticipantsResponse
	err      error
}

func (f fakeWebParticipantClient) ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListParticipantsResponse{}, nil
}

type fakeWebCharacterClient struct {
	response *statev1.ListCharactersResponse
	err      error
}

func (f fakeWebCharacterClient) ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListCharactersResponse{}, nil
}

type fakeWebSessionClient struct {
	response *statev1.ListSessionsResponse
	err      error
}

func (f fakeWebSessionClient) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

type fakeWebInviteClient struct {
	response *statev1.ListInvitesResponse
	err      error
}

func (f fakeWebInviteClient) ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListInvitesResponse{}, nil
}

type fakeWebAuthClient struct {
	mu       sync.Mutex
	sessions map[string]string
}

type countingWebAuthClient struct {
	*fakeWebAuthClient
	countMu            sync.Mutex
	getWebSessionCalls int
}

func newCountingWebAuthClient() *countingWebAuthClient {
	return &countingWebAuthClient{fakeWebAuthClient: newFakeWebAuthClient()}
}

func (f *countingWebAuthClient) GetWebSession(ctx context.Context, req *authv1.GetWebSessionRequest, opts ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	f.countMu.Lock()
	f.getWebSessionCalls++
	f.countMu.Unlock()
	return f.fakeWebAuthClient.GetWebSession(ctx, req, opts...)
}

func (f *countingWebAuthClient) GetWebSessionCalls() int {
	f.countMu.Lock()
	defer f.countMu.Unlock()
	return f.getWebSessionCalls
}

func newFakeWebAuthClient() *fakeWebAuthClient {
	return &fakeWebAuthClient{sessions: map[string]string{}}
}

func (f *fakeWebAuthClient) CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","rp":{"name":"web"},"user":{"id":"dXNlcg","name":"new@example.com","displayName":"new@example.com"},"pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`)}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-session", CredentialRequestOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","timeout":60000,"userVerification":"preferred"}}`)}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) CreateWebSession(_ context.Context, req *authv1.CreateWebSessionRequest, _ ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := "ws-1"
	f.sessions[id] = req.GetUserId()
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: id, UserId: req.GetUserId()}, User: &authv1.User{Id: req.GetUserId()}}, nil
}

func (f *fakeWebAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, ok := f.sessions[req.GetSessionId()]
	if !ok {
		return nil, context.Canceled
	}
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: req.GetSessionId(), UserId: userID}, User: &authv1.User{Id: userID}}, nil
}

func (f *fakeWebAuthClient) RevokeWebSession(_ context.Context, req *authv1.RevokeWebSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, req.GetSessionId())
	return &authv1.RevokeWebSessionResponse{}, nil
}

func (f fakeCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f fakeCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Id: "c1", Name: "Campaign"}}, nil
}

func (f fakeCampaignClient) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "created"}}, nil
}
