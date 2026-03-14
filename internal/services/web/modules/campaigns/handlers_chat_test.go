package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountCampaignGameRouteRendersNavbarBackButton(t *testing.T) {
	t.Parallel()

	m := New(configWithGatewayAndChatFallback(fakeGateway{items: []campaignapp.CampaignSummary{{
		ID:            "c1",
		Name:          "The Guildhouse",
		CoverImageURL: "/static/campaign-covers/abandoned_castle_courtyard.png",
	}}}, modulehandler.NewTestBase(), nil, "8086"))

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
		`data-campaign-game-page="true"`,
		`data-campaign-chat-title="The Guildhouse Game"`,
		`href="/app/campaigns/c1"`,
		`data-chat-fallback-port="8086"`,
		`id="game-transcript"`,
		`<textarea id="game-message-input"`,
		`maxlength="12000"`,
		`id="game-persona-select"`,
		`id="game-request-handoff"`,
		`id="campaign-game-bootstrap"`,
		`src="/static/campaign-chat.js"`,
		`class="navbar-start"`,
		`class="navbar-end"`,
		`class="btn btn-ghost"`,
		`Back to Campaign`,
		`class="grid grid-cols-1 gap-4 xl:grid-cols-[18rem_minmax(0,1fr)_18rem]"`,
		`class="card border border-base-300 bg-base-100 shadow-xl"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing chat page marker %q: %q", marker, body)
		}
	}
	// Invariant: drawer is removed — no drawer classes should be present.
	for _, absent := range []string{
		`drawer-toggle`,
		`drawer-side`,
		`chat-drawer-shell`,
	} {
		if strings.Contains(body, absent) {
			t.Fatalf("body unexpectedly contains removed drawer marker %q", absent)
		}
	}
	// Invariant: dedicated chat route must not render default app chrome shell wrappers.
	if strings.Contains(body, `id="main"`) || strings.Contains(body, `data-nav-item="true"`) {
		t.Fatalf("chat route unexpectedly rendered app chrome: %q", body)
	}
}

func TestMountCampaignGameRouteHTMXRedirectsToFullPage(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{items: []campaignapp.CampaignSummary{{ID: "c1", Name: "The Guildhouse"}}}, modulehandler.NewTestBase(), nil))
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
