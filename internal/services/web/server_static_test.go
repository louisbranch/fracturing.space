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

	h, err := newTestHandler(Config{
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
	body := rr.Body.String()
	for _, marker := range []string{
		`--app-scroll-offset`,
		`--app-fixed-header-height`,
		`scroll-padding-top: var(--app-scroll-offset);`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("theme.css missing scroll offset marker %q", marker)
		}
	}
}

func TestStaticPasskeyAuthScriptIncludesBusyStateGuard(t *testing.T) {
	t.Parallel()

	h, err := newTestHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/passkey-auth.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/javascript") && !strings.Contains(ct, "text/javascript") {
		t.Fatalf("content-type = %q, want javascript", ct)
	}

	body := rr.Body.String()
	for _, marker := range []string{
		`data-passkey-busy`,
		`data-button-loading-spinner='true'`,
		`spinner.className = "loading loading-spinner"`,
		`button.setAttribute("aria-busy", "true")`,
		`if (!markPasskeyBusy(loginForm, passkeyButton))`,
		`clearPasskeyBusy(registerForm, registerButton, shouldDisableRegisterButton(registerButton))`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("passkey-auth.js missing busy-state marker %q", marker)
		}
	}
	if strings.Contains(body, `button.classList.add("loading")`) {
		t.Fatalf("passkey-auth.js should not use button.loading class: %q", body)
	}
}

func TestStaticUsernameInputScriptPreservesSignupAvailabilityDuringBusyState(t *testing.T) {
	t.Parallel()

	h, err := newTestHandler(Config{
		Dependencies: newDefaultDependencyBundle(modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}}),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/static/username-input.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/javascript") && !strings.Contains(ct, "text/javascript") {
		t.Fatalf("content-type = %q, want javascript", ct)
	}

	body := rr.Body.String()
	for _, marker := range []string{
		`data-passkey-register-allowed`,
		`data-button-loading-spinner='true'`,
		`spinner.className = "loading loading-spinner"`,
		`form.getAttribute("data-passkey-busy") === "true"`,
		`syncButtonEnabledState()`,
		`button.setAttribute("data-passkey-register-allowed", enabled ? "true" : "false")`,
		`input.setAttribute("aria-controls", resultsEl.id)`,
		`resultsEl.setAttribute("role", "listbox")`,
		`input.addEventListener("keydown"`,
		`data-campaign-invite-search-result-active`,
		`document.addEventListener("mousedown"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("username-input.js missing signup busy-state marker %q", marker)
		}
	}
	if strings.Contains(body, `button.classList.add("loading")`) {
		t.Fatalf("username-input.js should not use button.loading class: %q", body)
	}
}

func TestStaticAppShellScriptIncludesHTMXErrorSwapContract(t *testing.T) {
	t.Parallel()

	h, err := newTestHandler(Config{
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

	h, err := newTestHandler(Config{
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
		`data-app-navbar`,
		`syncAppScrollOffset`,
		`--app-scroll-offset`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("app-shell.js missing route metadata marker %q", marker)
		}
	}
}

func TestStaticAppShellScriptOmitsImageFallbackContract(t *testing.T) {
	t.Parallel()

	h, err := newTestHandler(Config{
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
	cfg := defaultProtectedConfig(auth)
	cfg.Dependencies.Modules.Campaigns.CampaignClient = fakeCampaignClient{
		response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Remote"}}},
	}
	h, err := newTestHandler(cfg)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/c1/game", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); !strings.HasPrefix(got, "http://play.example.com/campaigns/c1?launch=") {
		t.Fatalf("Location = %q, want play host handoff", got)
	}
}
