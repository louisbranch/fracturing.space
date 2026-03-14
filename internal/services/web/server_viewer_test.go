package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/icons"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"google.golang.org/grpc"
)

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
	if !strings.Contains(body, `id="dashboard-root" hx-history="false"`) {
		t.Fatalf("body missing dashboard history opt-out")
	}
	assertPrimaryNavLinks(t, body)
}

func TestPrimaryNavigationOmitsUnavailableLinks(t *testing.T) {
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
	if !strings.Contains(body, `href="/app/notifications"`) {
		t.Fatalf("body missing stable notifications link: %q", body)
	}
	if strings.Contains(body, `href="/app/profile"`) {
		t.Fatalf("body unexpectedly includes unavailable profile link: %q", body)
	}
}

func TestPrimaryNavigationOmitsNotificationsLinkWhenNotificationsModuleUnavailable(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	cfg := defaultProtectedConfig(auth)
	cfg.Dependencies.Principal.NotificationClient = nil
	cfg.Dependencies.Modules.Notifications.NotificationClient = nil
	h, err := NewHandler(cfg)
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
		t.Fatalf("body unexpectedly includes notifications link: %q", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/notifications", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("notifications route status = %d, want %d", rr.Code, http.StatusNotFound)
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
	notificationReadIconHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_NOTIFICATION)) + `"`
	if !strings.Contains(body, notificationReadIconHref) {
		t.Fatalf("body missing read notifications icon %q", notificationReadIconHref)
	}
	notificationUnreadIconHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_NOTIFICATION_UNREAD)) + `"`
	if strings.Contains(body, notificationUnreadIconHref) {
		t.Fatalf("body unexpectedly contains unread notifications icon %q", notificationUnreadIconHref)
	}
}

func TestPrimaryNavigationUsesUnreadNotificationIconWhenUserHasUnread(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	cfg := defaultProtectedConfig(auth)
	notifClient := fakeWebNotificationClient{
		unreadResp: &notificationsv1.GetUnreadNotificationStatusResponse{HasUnread: true, UnreadCount: 2},
	}
	cfg.Dependencies.Principal.NotificationClient = notifClient
	cfg.Dependencies.Modules.Notifications.NotificationClient = notifClient
	h, err := NewHandler(cfg)
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
	notificationUnreadIconHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_NOTIFICATION_UNREAD)) + `"`
	if !strings.Contains(body, notificationUnreadIconHref) {
		t.Fatalf("body missing unread notifications icon %q", notificationUnreadIconHref)
	}
	notificationReadIconHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_NOTIFICATION)) + `"`
	if strings.Contains(body, notificationReadIconHref) {
		t.Fatalf("body unexpectedly contains read notifications icon %q", notificationReadIconHref)
	}
}

func TestPrimaryNavigationFallsBackToReadNotificationIconWhenUnreadLookupFails(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	cfg := defaultProtectedConfig(auth)
	notifClient := fakeWebNotificationClient{unreadErr: context.Canceled}
	cfg.Dependencies.Principal.NotificationClient = notifClient
	cfg.Dependencies.Modules.Notifications.NotificationClient = notifClient
	h, err := NewHandler(cfg)
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
	notificationReadIconHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_NOTIFICATION)) + `"`
	if !strings.Contains(body, notificationReadIconHref) {
		t.Fatalf("body missing read notifications icon %q", notificationReadIconHref)
	}
	notificationUnreadIconHref := `href="#` + icons.LucideSymbolID(icons.LucideNameOrDefault(commonv1.IconId_ICON_ID_NOTIFICATION_UNREAD)) + `"`
	if strings.Contains(body, notificationUnreadIconHref) {
		t.Fatalf("body unexpectedly contains unread notifications icon %q", notificationUnreadIconHref)
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
	if !strings.Contains(body, `<title>Profile | Fracturing.Space</title>`) {
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

	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{},
			modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}},
		),
	})
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

	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{},
			modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}},
		),
	})
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
		`Welcome to Fracturing.Space`,
		`lg:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]`,
		`lg:divider-horizontal`,
		`id="register-username"`,
		`id="login-username"`,
		`Create Account With Passkey`,
		`Log In With Passkey`,
		`/passkeys/login/start`,
		`/passkeys/login/finish`,
		`/passkeys/register/start`,
		`/passkeys/register/finish`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing auth contract marker %q", marker)
		}
	}
	if strings.Contains(body, `id="username"`) {
		t.Fatalf("body still renders legacy shared username input: %q", body)
	}
	for _, removed := range []string{
		`This becomes your public handle.`,
		`Username is required to find your account before passkey sign-in.`,
	} {
		if strings.Contains(body, removed) {
			t.Fatalf("body still renders removed auth helper copy %q", removed)
		}
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

func TestAppPageIncludesRouteMetadataAttribute(t *testing.T) {
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
	if !strings.Contains(body, `data-app-route-area="default"`) {
		t.Fatalf("body = %q, want default route area metadata", body)
	}
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

	social := &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "Rhea Vale", AvatarSetId: "avatar_set_v1", AvatarAssetId: "apothecary_journeyman"}}}
	assetBaseURL := "https://cdn.example.com/avatars"
	expectedAvatarURL := websupport.AvatarImageURL(assetBaseURL, "user", "user-1", "avatar_set_v1", "apothecary_journeyman", 40)
	auth := newFakeWebAuthClient()
	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "rhea", Locale: commonv1.Locale_LOCALE_EN_US}}}
	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{
				SessionClient: auth,
				AccountClient: account,
				SocialClient:  social,
				AssetBaseURL:  assetBaseURL,
			},
			modules.Dependencies{
				AssetBaseURL: assetBaseURL,
				PublicAuth:   modules.PublicAuthDependencies{AuthClient: auth},
				Profile:      modules.ProfileDependencies{SocialClient: social},
				Settings: modules.SettingsDependencies{
					SocialClient:     social,
					AccountClient:    account,
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
				Campaigns: modules.CampaignDependencies{CampaignClient: defaultCampaignClient(), CommunicationClient: defaultCommunicationClient()},
			},
		),
	})
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
	if !account.getProfileCalled {
		t.Fatalf("expected account profile lookup")
	}
	for _, marker := range []string{
		`src="` + expectedAvatarURL + `"`,
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

func TestAppPageUserDropdownProfileUsesAuthUsernameWhenSocialProfileHasNoUsername(t *testing.T) {
	t.Parallel()

	social := &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "Rhea Vale"}}}
	auth := newFakeWebAuthClient()
	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "rhea", Locale: commonv1.Locale_LOCALE_EN_US}}}
	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{
				SessionClient: auth,
				AccountClient: account,
				SocialClient:  social,
				AssetBaseURL:  "https://cdn.example.com/avatars",
			},
			modules.Dependencies{
				AssetBaseURL: "https://cdn.example.com/avatars",
				PublicAuth:   modules.PublicAuthDependencies{AuthClient: auth},
				Profile:      modules.ProfileDependencies{SocialClient: social},
				Settings: modules.SettingsDependencies{
					SocialClient:     social,
					AccountClient:    account,
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
				Campaigns: modules.CampaignDependencies{CampaignClient: defaultCampaignClient(), CommunicationClient: defaultCommunicationClient()},
			},
		),
	})
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
	if !strings.Contains(body, `href="/u/rhea"`) {
		t.Fatalf("body missing auth-backed public profile link: %q", body)
	}
}

func TestAppPageUsesDeterministicAvatarWhenProfileHasNoAssetSelection(t *testing.T) {
	t.Parallel()

	social := &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "Rhea Vale"}}}
	assetBaseURL := "https://cdn.example.com/avatars"
	expectedAvatarURL := websupport.AvatarImageURL(assetBaseURL, "user", "user-1", "", "", 40)
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{
				SessionClient: auth,
				SocialClient:  social,
				AssetBaseURL:  assetBaseURL,
			},
			modules.Dependencies{
				AssetBaseURL: assetBaseURL,
				PublicAuth:   modules.PublicAuthDependencies{AuthClient: auth},
				Profile:      modules.ProfileDependencies{SocialClient: social},
				Settings: modules.SettingsDependencies{
					SocialClient:     social,
					AccountClient:    &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}},
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
				Campaigns: modules.CampaignDependencies{CampaignClient: defaultCampaignClient(), CommunicationClient: defaultCommunicationClient()},
			},
		),
	})
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
		`class="relative z-1 h-full w-full object-cover rounded-full"`,
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
	getProfileCalls  int
	lastUpdateReq    *authv1.UpdateProfileRequest
	updateErr        error
}

type fakeCredentialClient struct{}
type fakeAgentClient struct{}

func (f *fakeAccountClient) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	f.getProfileCalled = true
	f.getProfileCalls++
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

func (f *fakeSocialClient) SearchUsers(context.Context, *socialv1.SearchUsersRequest, ...grpc.CallOption) (*socialv1.SearchUsersResponse, error) {
	return &socialv1.SearchUsersResponse{}, nil
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

func (fakeCredentialClient) ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	return &aiv1.ListCredentialsResponse{}, nil
}

func (fakeCredentialClient) CreateCredential(context.Context, *aiv1.CreateCredentialRequest, ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	return &aiv1.CreateCredentialResponse{}, nil
}

func (fakeCredentialClient) RevokeCredential(context.Context, *aiv1.RevokeCredentialRequest, ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	return &aiv1.RevokeCredentialResponse{}, nil
}

func (fakeAgentClient) ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	return &aiv1.ListAgentsResponse{}, nil
}

func (fakeAgentClient) ListProviderModels(context.Context, *aiv1.ListProviderModelsRequest, ...grpc.CallOption) (*aiv1.ListProviderModelsResponse, error) {
	return &aiv1.ListProviderModelsResponse{}, nil
}

func (fakeAgentClient) CreateAgent(context.Context, *aiv1.CreateAgentRequest, ...grpc.CallOption) (*aiv1.CreateAgentResponse, error) {
	return &aiv1.CreateAgentResponse{}, nil
}
