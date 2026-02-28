package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountCampaignGameRouteRendersDedicatedDrawerChrome(t *testing.T) {
	t.Parallel()

	m := NewExperimentalWithGateway(fakeGateway{items: []CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), "8086", nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignGame("c1"), nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-chat-page="true"`,
		`class="drawer lg:drawer-open min-h-[100dvh] campaign-chat-drawer"`,
		`class="drawer-side is-drawer-close:overflow-visible"`,
		`class="chat-drawer-shell flex min-h-full flex-col items-start border-e border-base-300 bg-base-200"`,
		`class="drawer-overlay chat-drawer-overlay lg:hidden"`,
		`data-campaign-chat-title="The Guildhouse Game"`,
		`class="px-2 text-lg font-bold"`,
		`href="/app/campaigns/c1"`,
		`data-chat-fallback-port="8086"`,
		`id="chat-messages"`,
		`src="/static/campaign-chat.js"`,
		`class="chat-drawer-icon-open size-5"`,
		`class="chat-drawer-icon-close size-5"`,
		`class="chat-drawer-link-label"`,
		`panel-left-open`,
		`panel-right-close`,
		`<symbol id="lucide-panel-left-open"`,
		`<symbol id="lucide-panel-right-close"`,
		`class="grid grid-cols-1 gap-4 lg:grid-cols-2"`,
		`class="card border border-base-300 bg-base-100 shadow-xl"`,
		`<span class="chat-drawer-link-label">Campaign</span>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing chat page marker %q: %q", marker, body)
		}
	}
	// Invariant: chat title should stay near the left toggle, not centered.
	if strings.Contains(body, `navbar-center`) {
		t.Fatalf("chat route unexpectedly centers navbar title: %q", body)
	}
	// Invariant: dedicated chat route must not render default app chrome shell wrappers.
	if strings.Contains(body, `id="main"`) || strings.Contains(body, `data-nav-item="true"`) {
		t.Fatalf("chat route unexpectedly rendered app chrome: %q", body)
	}
}

func TestMountCampaignGameRouteHTMXRedirectsToFullPage(t *testing.T) {
	t.Parallel()

	m := NewExperimentalWithGateway(fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}}, modulehandler.NewTestBase(), "", nil)
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignGame("c1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignGame("c1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignGame("c1"))
	}
}
