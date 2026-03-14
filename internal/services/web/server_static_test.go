package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
)

func TestStaticThemeServedByWeb(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/theme.css", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("content-type = %q, want text/css", ct)
	}
}

func TestStaticCampaignChatScriptServedByWeb(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/campaign-chat.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/javascript") && !strings.Contains(ct, "text/javascript") {
		t.Fatalf("content-type = %q, want javascript", ct)
	}
}

func TestStaticCampaignChatScriptIncludesAppHostFallbackLogic(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/campaign-chat.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		"replaceAppHostPrefix",
		"if (fallbackPort)",
		"host.indexOf(\"app.\") === 0",
		"addWSHostCandidate(chatHost + \":\" + fallbackPort)",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("chat script missing fallback marker %q", marker)
		}
	}
}

func TestStaticCampaignChatScriptPrefersFallbackPortForLocalhost(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/campaign-chat.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	idxFallback := strings.Index(body, "if (fallbackPort) {")
	if idxFallback < 0 {
		t.Fatalf("chat script missing fallback block")
	}
	idxResolvedGate := strings.Index(body, "if (!canUseLocalFallback || !fallbackPort) {")
	if idxResolvedGate < 0 {
		t.Fatalf("chat script missing resolved-host gate")
	}
	if idxFallback >= idxResolvedGate {
		t.Fatalf("fallback block should run before resolved-host gate")
	}
}

func TestStaticAppShellScriptIncludesHTMXErrorSwapContract(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/app-shell.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`document.addEventListener("htmx:beforeSwap"`,
		`event.detail.shouldSwap = true`,
		`event.detail.isError = false`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("app-shell.js missing htmx contract marker %q", marker)
		}
	}
}

func TestStaticAppShellScriptIncludesRouteMetadataContract(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/app-shell.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-app-route-area`,
		`campaignWorkspaceRouteArea`,
		`isCampaignWorkspaceMetadata`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("app-shell.js missing route metadata marker %q", marker)
		}
	}
}

func TestStaticAppShellScriptOmitsImageFallbackContract(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/app-shell.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`initAppImageFallbacks`,
		`syncAppImageStates`,
		`document.addEventListener("load"`,
		`document.addEventListener("error"`,
		`hideAppImageSkeleton`,
		`showAppImageSkeleton`,
		`target.style.display = "none"`,
	} {
		// Invariant: image fallback behavior is intentionally removed from app-shell.js.
		if strings.Contains(body, marker) {
			t.Fatalf("app-shell.js unexpectedly contains removed image fallback marker %q", marker)
		}
	}
}

func TestCampaignGamePageIsExposedOnDefaultCampaignSurface(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{
		ChatHTTPAddr: "localhost:8086",
		Dependencies: newDependencyBundle(
			PrincipalDependencies{SessionClient: auth},
			modules.Dependencies{
				PublicAuth: modules.PublicAuthDependencies{AuthClient: auth},
				Campaigns: modules.CampaignDependencies{
					CampaignClient:           fakeCampaignClient{response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Remote"}}}},
					CommunicationClient:      defaultCommunicationClient(),
					ParticipantClient:        defaultParticipantClient(),
					CharacterClient:          defaultCharacterClient(),
					DaggerheartContentClient: defaultDaggerheartContentClient(),
					DaggerheartAssetClient:   defaultDaggerheartAssetClient(),
					SessionClient:            defaultSessionClient(),
					InviteClient:             defaultInviteClient(),
					AuthClient:               auth,
					AuthorizationClient:      defaultAuthorizationClient(),
				},
			},
		),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/c1/game", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}
